package server

import (
	"log"
	"net/http"
	"os"
	"sync"
)

var (
	openapiOnce sync.Once
	openapiSpec []byte
	errOpenapi  error
)

func loadOpenapiSpec() {
	openapiSpec, errOpenapi = os.ReadFile("docs/openapi.yaml")
}

var scalarHTML = []byte(`<!doctype html>
<html>
<head>
  <title>Castellan API</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>
    body { margin: 0; }
  </style>
</head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  <script>
    Scalar.createApiReference('#app', {
      url: '/openapi.yaml',
      theme: 'default',
      hideDownloadButton: true,
    })
  </script>
</body>
</html>`)

func openapiHandler(w http.ResponseWriter, _ *http.Request) {
	openapiOnce.Do(loadOpenapiSpec)
	if errOpenapi != nil {
		http.Error(w, "openapi spec not found", http.StatusNotFound)

		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	if _, err := w.Write(openapiSpec); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func docsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write(scalarHTML); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
