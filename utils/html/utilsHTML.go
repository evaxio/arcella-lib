package html

import (
	"golang.org/x/net/html"
	"strings"
)

func HtmlToText(s string) string {
	var b strings.Builder
	entered := false
	inTag := ""
	domDocTest := html.NewTokenizer(strings.NewReader(s))
	for {
		tt := domDocTest.Next()
		switch {
		case tt == html.ErrorToken:
			return b.String()
		case tt == html.StartTagToken:
			token := domDocTest.Token()
			if token.Data == "script" || token.Data == "style" {
				inTag = token.Data
			}
		case tt == html.EndTagToken:
			token := domDocTest.Token()
			if token.Data == inTag {
				inTag = ""
			}
			if inTag == "" && !entered {
				b.WriteString("\n")
				entered = true
			}
		case tt == html.TextToken:
			if inTag != "" {
				continue
			}
			TxtContent := strings.TrimSpace(html.UnescapeString(string(domDocTest.Text())))
			if len(TxtContent) > 0 {
				b.WriteString(TxtContent)
				entered = false
			}
		}
	}
}
