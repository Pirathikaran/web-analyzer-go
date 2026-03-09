package analyzer

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

const MaxLinks = 500

func collectLinks(doc *html.Node, base *url.URL) (internal, external []string) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if len(internal)+len(external) >= MaxLinks {
			return
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			if href := attrVal(n, "href"); href != "" {
				if abs, ok := resolveURL(base, href); ok {
					if isInternal(base, abs) {
						internal = append(internal, abs)
					} else {
						external = append(external, abs)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return internal, external
}

func resolveURL(base *url.URL, href string) (string, bool) {
	href = strings.TrimSpace(href)
	switch {
	case href == "" || href == "#":
		return "", false
	case strings.HasPrefix(href, "#"):
		return "", false
	case strings.HasPrefix(strings.ToLower(href), "mailto:"):
		return "", false
	case strings.HasPrefix(strings.ToLower(href), "javascript:"):
		return "", false
	case strings.HasPrefix(strings.ToLower(href), "tel:"):
		return "", false
	}

	ref, err := url.Parse(href)
	if err != nil {
		return "", false
	}
	abs := base.ResolveReference(ref)
	abs.Fragment = ""
	return abs.String(), true
}

func isInternal(base *url.URL, target string) bool {
	t, err := url.Parse(target)
	if err != nil {
		return false
	}
	return strings.EqualFold(t.Hostname(), base.Hostname())
}