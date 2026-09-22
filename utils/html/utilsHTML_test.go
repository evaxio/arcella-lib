package html

import (
	"strings"
	"testing"
)

func TestHtmlToText(t *testing.T) {
	out := HtmlToText("<p>hello</p>")
	if !strings.Contains(out, "hello") {
		t.Errorf("output = %q, want to contain hello", out)
	}
}

func TestHtmlToTextDropsScript(t *testing.T) {
	out := HtmlToText("<script>bad()</script><p>ok</p>")
	if !strings.Contains(out, "ok") {
		t.Errorf("output = %q, want to contain ok", out)
	}
	if strings.Contains(out, "bad") {
		t.Errorf("output = %q, script content must be dropped", out)
	}
}

func TestHtmlToTextUnescapes(t *testing.T) {
	out := HtmlToText("<p>a&amp;b</p>")
	if !strings.Contains(out, "a&b") {
		t.Errorf("output = %q, want a&b", out)
	}
}

func TestHtmlToTextNested(t *testing.T) {
	out := HtmlToText("<div>a<b>b</b></div>")
	if !strings.Contains(out, "a") || !strings.Contains(out, "b") {
		t.Errorf("output = %q, want a and b", out)
	}
}
