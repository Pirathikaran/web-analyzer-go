package analyzer

import (
	"strings"

	"golang.org/x/net/html"
)

// detectHTMLVersion inspects the document's DOCTYPE node to determine the HTML version.
func detectHTMLVersion(doc *html.Node) string {
	for n := doc.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.DoctypeNode {
			return classifyDoctype(n)
		}
	}
	return "Unknown"
}

func classifyDoctype(n *html.Node) string {
	name := strings.ToLower(strings.TrimSpace(n.Data))
	public := strings.ToLower(attrVal(n, "public"))
	system := strings.ToLower(attrVal(n, "system"))

	switch {
	case name == "html" && public == "" && system == "":
		return "HTML5"
	case strings.Contains(public, "xhtml 1.1"):
		return "XHTML 1.1"
	case strings.Contains(public, "xhtml 1.0 strict"):
		return "XHTML 1.0 Strict"
	case strings.Contains(public, "xhtml 1.0 transitional"):
		return "XHTML 1.0 Transitional"
	case strings.Contains(public, "xhtml 1.0 frameset"):
		return "XHTML 1.0 Frameset"
	case strings.Contains(public, "html 4.01") && strings.Contains(public, "strict"):
		return "HTML 4.01 Strict"
	case strings.Contains(public, "html 4.01") && strings.Contains(public, "transitional"):
		return "HTML 4.01 Transitional"
	case strings.Contains(public, "html 4.01") && strings.Contains(public, "frameset"):
		return "HTML 4.01 Frameset"
	case strings.Contains(public, "html 4.0"):
		return "HTML 4.0"
	case strings.Contains(public, "html 3.2"):
		return "HTML 3.2"
	case strings.Contains(public, "html 2.0"):
		return "HTML 2.0"
	case name == "html":
		return "HTML (version unknown)"
	default:
		return "Unknown"
	}
}

// attrVal returns the value of the named attribute, or "" if absent.
func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}
