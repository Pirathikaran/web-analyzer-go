package analyzer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

const (
	maxResponseBodyBytes  = 10 * 1024 * 1024 // 10 MB
	linkCheckConcurrency  = 10
)

var urlPattern = regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)

type Result struct {
	URL               string
	HTMLVersion       string
	Title             string
	Headings          map[string]int
	InternalLinks     int
	ExternalLinks     int
	InaccessibleLinks int
	HasLoginForm      bool
}

type Analyzer struct {
	client    *http.Client
	logger    *slog.Logger
	globalSem chan struct{} // global cap on concurrent outbound link-check requests
}

func New(client *http.Client, logger *slog.Logger, globalSem chan struct{}) *Analyzer {
	return &Analyzer{client: client, logger: logger, globalSem: globalSem}
}

func ValidateURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL must not be empty")
	}
	if !urlPattern.MatchString(rawURL) {
		return fmt.Errorf("invalid URL format: %q", rawURL)
	}
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return errors.New("invalid url scheme")
	}
	return nil
}

func (a *Analyzer) Analyze(ctx context.Context, rawURL string) (*Result, error) {
	a.logger.InfoContext(ctx, "starting analysis", "url", rawURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "web-analyzer/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch URL: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			a.logger.WarnContext(ctx, "failed to close response body", "err", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{Code: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	}

	doc, err := html.Parse(io.LimitReader(resp.Body, maxResponseBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}

	base, _ := url.Parse(rawURL)

	result := &Result{
		URL:      rawURL,
		Headings: make(map[string]int),
	}

	result.HTMLVersion = detectHTMLVersion(doc)
	result.Title = extractTitle(doc)
	collectHeadings(doc, result.Headings)
	result.HasLoginForm = hasLoginForm(doc)

	internal, external := collectLinks(doc, base)
	result.InternalLinks = len(internal)
	result.ExternalLinks = len(external)

	allLinks := append(internal, external...)
	result.InaccessibleLinks = a.checkInaccessibleLinks(ctx, allLinks)

	a.logger.InfoContext(ctx, "analysis complete",
		"url", rawURL,
		"html_version", result.HTMLVersion,
		"internal_links", result.InternalLinks,
		"external_links", result.ExternalLinks,
		"inaccessible", result.InaccessibleLinks,
	)

	return result, nil
}

func (a *Analyzer) checkInaccessibleLinks(ctx context.Context, links []string) int {
	seen := make(map[string]struct{}, len(links))
	for _, l := range links {
		seen[l] = struct{}{}
	}
	uniqueLinks := make([]string, 0, len(seen))
	for l := range seen {
		uniqueLinks = append(uniqueLinks, l)
	}

	if len(uniqueLinks) == 0 {
		return 0
	}

	linkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	type checkResult struct {
		url        string
		accessible bool
	}

	work := make(chan string, len(uniqueLinks))
	for _, l := range uniqueLinks {
		work <- l
	}
	close(work)

	results := make(chan checkResult, len(uniqueLinks))

	concurrency := linkCheckConcurrency
	if len(uniqueLinks) < concurrency {
		concurrency = len(uniqueLinks)
	}

	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for l := range work {
				results <- checkResult{url: l, accessible: a.isLinkAccessible(linkCtx, l)}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	inaccessibleURLs := make(map[string]struct{})
	for r := range results {
		if !r.accessible {
			inaccessibleURLs[r.url] = struct{}{}
		}
	}

	inaccessible := 0
	for _, l := range links {
		if _, bad := inaccessibleURLs[l]; bad {
			inaccessible++
		}
	}
	return inaccessible
}

func (a *Analyzer) isLinkAccessible(ctx context.Context, url string) bool {
	select {
	case a.globalSem <- struct{}{}:
		defer func() { <-a.globalSem }()
	case <-ctx.Done():
		return false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return false
	}
	req.Header.Set("User-Agent", "web-analyzer/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return false
	}
	if err := resp.Body.Close(); err != nil {
		a.logger.Warn("failed to close HEAD response body", "err", err)
	}

	if resp.StatusCode == http.StatusMethodNotAllowed {
		req2, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false
		}
		req2.Header.Set("User-Agent", "web-analyzer/1.0")
		resp2, err := a.client.Do(req2)
		if err != nil {
			return false
		}
		if err := resp2.Body.Close(); err != nil {
			a.logger.Warn("failed to close GET response body", "err", err)
		}
		return resp2.StatusCode < 400
	}

	return resp.StatusCode < 400
}

type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	msg := e.Message
	if msg == "" {
		switch e.Code {
		case 999:
			msg = "request blocked by the target site (bot protection)"
		default:
			msg = "unexpected status code"
		}
	}
	return fmt.Sprintf("HTTP %d: %s", e.Code, msg)
}