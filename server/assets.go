package server

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"mime"
	"os"
	"path"
	"strings"
)

const (
	webAssetPrefix   = "/"
	katexAssetBase   = webAssetPrefix + "katex/"
	builtinAssetBase = "builtin/"
)

// builtinAssets holds the frontend resources that the engine owns. Templates
// do not need to reference them; they are injected only into pages with math.
//
//go:embed builtin
var builtinAssets embed.FS

func init() {
	// Not registered by every system mime type database.
	_ = mime.AddExtensionType(".woff2", "font/woff2")
}

// webAssetName maps a URL path under katexAssetBase to a path in the embedded
// file system.
func webAssetName(urlPath string) (string, bool) {
	rel := strings.TrimPrefix(urlPath, webAssetPrefix)
	if rel == urlPath || !fs.ValidPath(rel) {
		return "", false
	}
	return builtinAssetBase + rel, true
}

// routeWebAsset resolves uri to an embedded asset, mirroring notesRouter.Route.
// A missing or invalid path returns os.ErrNotExist.
func routeWebAsset(uri string) (content []byte, mimeType string, err error) {
	name, ok := webAssetName(uri)
	if !ok {
		return nil, "", os.ErrNotExist
	}
	content, err = fs.ReadFile(builtinAssets, name)
	if err != nil {
		return nil, "", os.ErrNotExist
	}
	return content, mime.TypeByExtension(path.Ext(name)), nil
}

// injectMathAssets adds KaTeX's stylesheet and renderer to a page that contains
// rendered math. It leaves pages without math untouched and does not duplicate
// assets a template already references.
func injectMathAssets(page []byte) []byte {
	if !bytes.Contains(page, []byte(`class="math math-`)) {
		return page
	}

	lower := bytes.ToLower(page)
	var head strings.Builder
	if !bytes.Contains(lower, []byte("katex.min.css")) {
		fmt.Fprintf(&head, `<link rel="stylesheet" href="%skatex.min.css">`, katexAssetBase)
	}
	if !bytes.Contains(lower, []byte(katexAssetBase+"math.css")) {
		fmt.Fprintf(&head, `<link rel="stylesheet" href="%smath.css">`, katexAssetBase)
	}

	var body strings.Builder
	if !bytes.Contains(lower, []byte("katex.min.js")) {
		fmt.Fprintf(&body, `<script src="%skatex.min.js"></script>`, katexAssetBase)
	}
	if !bytes.Contains(lower, []byte(katexAssetBase+"math.js")) {
		fmt.Fprintf(&body, `<script src="%smath.js"></script>`, katexAssetBase)
	}

	if head.Len() > 0 {
		page = insertHTML(page, "</head>", head.String(), false)
	}
	if body.Len() > 0 {
		page = insertHTML(page, "</body>", body.String(), true)
	}
	return page
}

// insertHTML inserts snippet before the first case-insensitive occurrence of
// tag. If tag is missing, snippet is prepended (or appended, depending on
// appendIfMissing).
func insertHTML(page []byte, tag string, snippet string, appendIfMissing bool) []byte {
	idx := bytes.Index(bytes.ToLower(page), []byte(tag))
	out := make([]byte, 0, len(page)+len(snippet))
	if idx < 0 {
		if appendIfMissing {
			out = append(out, page...)
			out = append(out, snippet...)
		} else {
			out = append(out, snippet...)
			out = append(out, page...)
		}
		return out
	}
	out = append(out, page[:idx]...)
	out = append(out, snippet...)
	out = append(out, page[idx:]...)
	return out
}
