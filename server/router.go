package server

import (
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/Streamlet/notesite/note"
)

func newRouter(noteRoot string, templateRoot string) (http.Handler, error) {
	notesRouter, err := note.NewRouter(noteRoot, templateRoot)
	if err != nil {
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		content, mimeType, err := notesRouter.Route(r.URL.Path)
		if err != nil && os.IsNotExist(err) {
			content, mimeType, err = routeWebAsset(r.URL.Path)
			if err == nil {
				w.Header().Set("Cache-Control", "public, max-age=86400")
			}
		}
		if err != nil {
			if os.IsNotExist(err) {
				log.Println(r.RequestURI, "404:", err.Error())
				w.WriteHeader(http.StatusNotFound)
				return
			} else {
				log.Println(r.RequestURI, "500:", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		}
		if mimeType == "text/html" {
			content = injectMathAssets(content)
		}
		if mimeType == "" {
			if urlParts, err := url.Parse(r.RequestURI); err == nil {
				ext := filepath.Ext(urlParts.Path)
				if mimeType := mime.TypeByExtension(ext); mimeType != "" {
					w.Header().Add("Content-Type", mimeType)
				} else {
					w.Header().Add("Content-Type", "application/octet-stream")
				}
			}
		} else {
			w.Header().Set("Content-Type", mimeType)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	})

	return mux, nil
}
