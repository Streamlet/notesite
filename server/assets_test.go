package server

import (
	"os"
	"strings"
	"testing"
)

func TestInjectMathAssets(t *testing.T) {
	const mathPage = `<html><head><title>math</title></head><body><p><span class="math math-inline">\(x\)</span></p></body></html>`

	t.Run("math page", func(t *testing.T) {
		got := string(injectMathAssets([]byte(mathPage)))
		for _, want := range []string{
			`href="` + katexAssetBase + `katex.min.css">`,
			`href="` + katexAssetBase + `math.css">`,
			`src="` + katexAssetBase + `katex.min.js">`,
			`src="` + katexAssetBase + `math.js">`,
		} {
			if !strings.Contains(got, want) {
				t.Errorf("injected page should contain %q:\n%s", want, got)
			}
		}
		if strings.Index(got, "katex.min.css") > strings.Index(got, "</head>") {
			t.Errorf("stylesheet should be injected into head:\n%s", got)
		}
		if strings.Index(got, "katex.min.js") > strings.Index(got, "</body>") {
			t.Errorf("scripts should be injected before </body>:\n%s", got)
		}
	})

	t.Run("page without math", func(t *testing.T) {
		page := `<html><head></head><body><p>plain</p></body></html>`
		if got := string(injectMathAssets([]byte(page))); got != page {
			t.Errorf("page without math changed:\n%s", got)
		}
	})

	t.Run("uppercase tags", func(t *testing.T) {
		page := `<HTML><HEAD></HEAD><BODY><span class="math math-inline">\(x\)</span></BODY></HTML>`
		got := string(injectMathAssets([]byte(page)))
		if !strings.Contains(got, "</HEAD>") || !strings.Contains(got, "</BODY>") {
			t.Fatalf("original tags should be preserved:\n%s", got)
		}
		if strings.Index(got, "katex.min.css") > strings.Index(got, "</HEAD>") {
			t.Errorf("stylesheet should be inserted before </HEAD>:\n%s", got)
		}
		if strings.Index(got, "katex.min.js") > strings.Index(got, "</BODY>") {
			t.Errorf("scripts should be inserted before </BODY>:\n%s", got)
		}
	})

	t.Run("existing katex is not duplicated", func(t *testing.T) {
		page := `<html><head><link rel="stylesheet" href="/vendor/katex.min.css"></head><body>` +
			`<span class="math math-inline">\(x\)</span><script src="/vendor/katex.min.js"></script></body></html>`
		got := string(injectMathAssets([]byte(page)))
		if strings.Contains(got, katexAssetBase+"katex.min.css") || strings.Contains(got, katexAssetBase+"katex.min.js") {
			t.Errorf("existing KaTeX should not be duplicated:\n%s", got)
		}
		if !strings.Contains(got, katexAssetBase+"math.css") || !strings.Contains(got, katexAssetBase+"math.js") {
			t.Errorf("notesite math assets should still be injected:\n%s", got)
		}
	})

	t.Run("missing head and body", func(t *testing.T) {
		page := `<span class="math math-inline">\(x\)</span>`
		got := string(injectMathAssets([]byte(page)))
		if !strings.HasPrefix(got, `<link rel="stylesheet"`) {
			t.Errorf("stylesheet should be prepended when head is missing:\n%s", got)
		}
		if !strings.HasSuffix(got, "</script>") {
			t.Errorf("scripts should be appended when body is missing:\n%s", got)
		}
	})
}

func TestRouteWebAsset(t *testing.T) {
	tests := []struct {
		path        string
		found       bool
		contentType string
	}{
		{katexAssetBase + "katex.min.js", true, "javascript"},
		{katexAssetBase + "katex.min.css", true, "text/css"},
		{katexAssetBase + "math.js", true, "javascript"},
		{katexAssetBase + "fonts/KaTeX_Main-Regular.woff2", true, "font/woff2"},
		{katexAssetBase, false, ""},
		{katexAssetBase + "missing.js", false, ""},
		{katexAssetBase + "../router.go", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			content, mimeType, err := routeWebAsset(tt.path)
			if !tt.found {
				if !os.IsNotExist(err) {
					t.Fatalf("err = %v, want os.ErrNotExist", err)
				}
				if content != nil || mimeType != "" {
					t.Fatalf("missing asset returned content or mime type")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(content) == 0 {
				t.Error("asset content should not be empty")
			}
			if !strings.Contains(mimeType, tt.contentType) {
				t.Errorf("mimeType = %q, want it to contain %q", mimeType, tt.contentType)
			}
		})
	}
}
