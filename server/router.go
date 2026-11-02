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
		if err != nil {
			if os.IsNotExist(err) {
				log.Println(r.RequestURI, "404:", err.Error())
				w.WriteHeader(http.StatusNotFound)
			} else {
				log.Println(r.RequestURI, "500:", err.Error())
				w.WriteHeader(http.StatusInternalServerError)
			}
		} else {
			if mimeType == "" {
				if urlParts, err := url.Parse(r.RequestURI); err == nil {
					ext := filepath.Ext(urlParts.Path)
					if mimeType := mime.TypeByExtension(ext); mimeType != "" {
						w.Header().Add("Content-Type", mimeType)
					} else {
						w.Header().Add("Content-Type", "application/octet-stream")
					}
				}
			}
			w.WriteHeader(http.StatusOK)
		}
		_, _ = w.Write(content)
	})

	return mux, nil
}
