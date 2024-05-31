//go:build dev

package app

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

var AppFS http.FileSystem

func init() {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
 		os.Stderr.WriteString("No caller information found")
  	os.Exit(1)
  }
  AppFS = http.Dir(filepath.Dir(filename))
}