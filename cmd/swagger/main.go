package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
)

//go:embed swagger-ui/*
var swaggerUI embed.FS

func main() {
	port := os.Getenv("SWAGGER_PORT")
	if port == "" {
		port = "18735"
	}

	// Load the swagger spec
	swaggerSpec, err := os.ReadFile("gen/api.swagger.json")
	if err != nil {
		log.Fatalf("Failed to read swagger spec: %v", err)
	}

	// Pretty print the JSON
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(swaggerSpec, &prettyJSON); err == nil {
		prettySpec, _ := json.MarshalIndent(prettyJSON, "", "  ")
		swaggerSpec = prettySpec
	}

	// Create a custom handler that serves the swagger UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(getSwaggerUIHTML()))
	})

	// Serve the swagger spec
	http.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(swaggerSpec)
	})

	// Serve the swagger-ui static files
	subFS, err := fs.Sub(swaggerUI, "swagger-ui")
	if err != nil {
		log.Fatalf("Failed to create sub filesystem: %v", err)
	}
	http.Handle("/swagger-ui/", http.StripPrefix("/swagger-ui/", http.FileServer(http.FS(subFS))))

	fmt.Printf("Swagger UI running at http://localhost:%s\n", port)
	fmt.Printf("API Spec available at http://localhost:%s/swagger.json\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getSwaggerUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>POS API - Swagger UI</title>
  <link rel="stylesheet" type="text/css" href="/swagger-ui/swagger-ui.css" />
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="/swagger-ui/swagger-ui-bundle.js"></script>
  <script src="/swagger-ui/swagger-ui-standalone-preset.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger.json",
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset
      ],
      layout: "StandaloneLayout"
    })
  </script>
</body>
</html>`
}
