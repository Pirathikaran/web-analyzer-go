package analyzer

import (
	"strings"

	"golang.org/x/net/html"
)

var headingTags = map[string]bool{
	"h1": true, "h2": true, "h3": true,
	"h4": true, "h5": true, "h6": true,
}

func extractTitle(doc *html.Node) string {
	var title string
	var walk func(*html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "title" {
			title = nodeText(n)
			return true // stop
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(doc)
	return strings.TrimSpace(title)
}

func collectHeadings(doc *html.Node, counts map[string]int) {
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && headingTags[n.Data] {
			counts[n.Data]++
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
}

func hasLoginForm(doc *html.Node) bool {
	var check func(*html.Node) bool
	check = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "form" {
			if formHasPasswordInput(n) {
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if check(c) {
				return true
			}
		}
		return false
	}
	return check(doc)
}

func formHasPasswordInput(form *html.Node) bool {
	var walk func(*html.Node) bool
	walk = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "input" {
			if strings.EqualFold(attrVal(n, "type"), "password") {
				return true
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if walk(c) {
				return true
			}
		}
		return false
	}
	return walk(form)
}

func nodeText(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return sb.String()
}
