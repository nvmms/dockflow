package webapi

import (
	_ "embed"
	"net/http"
)

//go:embed openapi.json
var openAPISpec []byte

func serveOpenAPI(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(openAPISpec)
}
