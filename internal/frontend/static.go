package frontend

import (
	"io/fs"
	"net/http"

	"embed"
)

//go:embed dist/*
var Static embed.FS

func StaticHandler() http.Handler {
	sub, _ := fs.Sub(Static, "dist")
	return http.StripPrefix("/", http.FileServerFS(sub))
}
