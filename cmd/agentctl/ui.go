package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var embeddedUIFS embed.FS

func uiHandler() http.Handler {
	sub, err := fs.Sub(embeddedUIFS, "ui")
	if err != nil {
		return http.NotFoundHandler()
	}
	return http.FileServer(http.FS(sub))
}

func serveUIIndex(w http.ResponseWriter, r *http.Request) {
	data, err := embeddedUIFS.ReadFile("ui/index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
