//go:build !dev

package app

import (
	"embed"
	"net/http"
)

//go:embed components/* styles/*
//go:embed index.html
var embedAppFS embed.FS
var AppFS = http.FS(embedAppFS)
