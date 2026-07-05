package server

import (
	"net/http"
	"os"
	"sync"
)

var (
	openapiOnce sync.Once
	openapiSpec []byte
	openapiErr  error
)

func loadOpenapiSpec() {
	openapiSpec, openapiErr = os.ReadFile("docs/openapi.yaml")
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
	if openapiErr != nil {
		http.Error(w, "openapi spec not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write(openapiSpec)
}

func docsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(scalarHTML)
}
