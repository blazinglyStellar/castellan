package server

import (
	_ "embed"
	"fmt"
	"html"
	"log"
	"net/http"
)

//go:embed openapi.yaml
var openapiSpec []byte

const scalarHTMLTmpl = `<!doctype html>
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
      servers: [{ url: '%s' }],
      theme: 'default',
      hideDownloadButton: true,
    })
  </script>
</body>
</html>`

func openapiHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/x-yaml")
	if _, err := w.Write(openapiSpec); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	// Extract base URL from request (scheme + Host)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := html.EscapeString(scheme + "://" + r.Host)

	// Generate HTML with base URL embedded
	htmlStr := fmt.Sprintf(scalarHTMLTmpl, baseURL)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(htmlStr)); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
