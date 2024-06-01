//go:build !dev

package app

import (
	"embed"
	"net/http"
)

//go:embed components/* internals/* services/* styles/*
//go:embed index.html favicon.ico
var embedAppFS embed.FS
var AppFS = http.FS(embedAppFS)
