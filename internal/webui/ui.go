package webui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"
)

//go:embed dist
var bundle embed.FS

func Assets() http.Handler {
	assets, err := fs.Sub(bundle, "dist")
	if err != nil {
		panic(err)
	}
	files := http.FileServerFS(assets)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(w, r)
	})
}

func Serve(w http.ResponseWriter, r *http.Request) {
	index, err := bundle.ReadFile("dist/index.html")
	if err != nil {
		http.Error(w, "UI bundle unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, "index.html", time.Time{}, bytes.NewReader(index))
}
