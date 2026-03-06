package analyzer

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
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
	client *http.Client
	logger *slog.Logger
}

func New(client *http.Client, logger *slog.Logger) *Analyzer {
	return &Analyzer{client: client, logger: logger}
}

func ValidateURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("URL must not be empty")
	}
	if !urlPattern.MatchString(rawURL) {
		return fmt.Errorf("invalid URL format: %q", rawURL)
	}
	if _, err := url.ParseRequestURI(rawURL); err != nil {
		return fmt.Errorf("URL parse error: %w", err)
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
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{Code: resp.StatusCode, Message: http.StatusText(resp.StatusCode)}
	}

	doc, err := html.Parse(resp.Body)
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
	type checkResult struct {
		accessible bool
	}

	results := make(chan checkResult, len(links))

	// Limit concurrency to avoid overwhelming targets.
	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup

	linkCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	for _, link := range links {
		wg.Add(1)
		go func(l string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			req, err := http.NewRequestWithContext(linkCtx, http.MethodHead, l, nil)
			if err != nil {
				results <- checkResult{accessible: false}
				return
			}
			req.Header.Set("User-Agent", "web-analyzer/1.0")

			resp, err := a.client.Do(req)
			if err != nil {
				results <- checkResult{accessible: false}
				return
			}
			resp.Body.Close()
			results <- checkResult{accessible: resp.StatusCode < 400}
		}(link)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	inaccessible := 0
	for r := range results {
		if !r.accessible {
			inaccessible++
		}
	}
	return inaccessible
}

type HTTPError struct {
	Code    int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s", e.Code, e.Message)
}
