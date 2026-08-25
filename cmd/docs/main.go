package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	port := os.Getenv("DOCS_PORT")
	if port == "" {
		port = "8080"
	}

	// Get the swagger file path (try enhanced version first, fallback to generated)
	swaggerPath := "docs/swagger-enhanced.json"
	if _, err := os.Stat(swaggerPath); os.IsNotExist(err) {
		swaggerPath = "gen/api.swagger.json"
	}
	if envPath := os.Getenv("SWAGGER_PATH"); envPath != "" {
		swaggerPath = envPath
	}

	// Serve the swagger spec dynamically
	http.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Read the swagger spec fresh each time
		swaggerSpec, err := os.ReadFile(swaggerPath)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "Failed to load swagger spec: %v"}`, err), http.StatusInternalServerError)
			return
		}

		// Pretty print the JSON
		var prettyJSON map[string]interface{}
		if err := json.Unmarshal(swaggerSpec, &prettyJSON); err == nil {
			prettySpec, _ := json.MarshalIndent(prettyJSON, "", "  ")
			w.Write(prettySpec)
		} else {
			w.Write(swaggerSpec)
		}
	})

	// Serve the swagger UI
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(getSwaggerUIHTML()))
			return
		}
		http.NotFound(w, r)
	})

	// Serve grpcurl examples
	http.HandleFunc("/examples", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(getExamplesHTML()))
	})

	// Redirect /docs to /
	http.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusMovedPermanently)
	})

	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "ok"}`))
	})

	fmt.Printf("POS API Documentation running at http://localhost:%s\n", port)
	fmt.Printf("Swagger UI: http://localhost:%s/\n", port)
	fmt.Printf("API Spec: http://localhost:%s/swagger.json\n", port)
	fmt.Printf("Examples: http://localhost:%s/examples\n", port)
	fmt.Printf("Swagger file path: %s\n", swaggerPath)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getSwaggerUIHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>POS API Documentation</title>
  <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui.css" />
  <style>
    html { box-sizing: border-box; overflow-y: scroll; }
    *, *:before, *:after { box-sizing: inherit; }
    body { margin: 0; background: #fafafa; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .topbar { display: none; }
    .info { margin: 20px; }
    .info h1 { color: #333; }
    .nav { background: #fff; padding: 10px 20px; border-bottom: 1px solid #e0e0e0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
    .nav a { margin-right: 20px; color: #1976d2; text-decoration: none; font-weight: 500; }
    .nav a:hover { text-decoration: underline; }
    .nav a.active { color: #0d47a1; border-bottom: 2px solid #1976d2; padding-bottom: 2px; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="/" class="active">Swagger UI</a>
    <a href="/examples">gRPC Examples</a>
    <a href="/swagger.json">Raw Spec</a>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.3/swagger-ui-standalone-preset.js"></script>
  <script>
    SwaggerUIBundle({
      url: "/swagger.json",
      dom_id: '#swagger-ui',
      deepLinking: true,
      presets: [
        SwaggerUIBundle.presets.apis,
        SwaggerUIStandalonePreset
      ],
      layout: "StandaloneLayout",
      docExpansion: "list",
      defaultModelsExpandDepth: -1
    })
  </script>
</body>
</html>`
}

func getExamplesHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>POS API - gRPC Examples</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 0; background: #fafafa; }
    .container { max-width: 1000px; margin: 0 auto; padding: 20px; }
    h1 { color: #333; margin-top: 20px; }
    h2 { color: #1976d2; margin-top: 30px; }
    .nav { background: #fff; padding: 10px 20px; border-bottom: 1px solid #e0e0e0; box-shadow: 0 2px 4px rgba(0,0,0,0.1); margin-bottom: 20px; }
    .nav a { margin-right: 20px; color: #1976d2; text-decoration: none; font-weight: 500; }
    .nav a:hover { text-decoration: underline; }
    .nav a.active { color: #0d47a1; border-bottom: 2px solid #1976d2; padding-bottom: 2px; }
    pre { background: #263238; color: #aed581; padding: 15px; border-radius: 8px; overflow-x: auto; font-size: 14px; }
    code { font-family: "Fira Code", "Consolas", monospace; }
    .endpoint { background: #fff; padding: 15px; margin: 10px 0; border-radius: 8px; border-left: 4px solid #1976d2; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
    .method { display: inline-block; padding: 2px 8px; border-radius: 4px; font-weight: bold; font-size: 12px; margin-right: 10px; }
    .method-post { background: #4caf50; color: white; }
    .method-get { background: #2196f3; color: white; }
    .method-put { background: #ff9800; color: white; }
    .method-delete { background: #f44336; color: white; }
    .method-patch { background: #9c27b0; color: white; }
    .section { margin-bottom: 40px; }
  </style>
</head>
<body>
  <div class="nav">
    <a href="/">Swagger UI</a>
    <a href="/examples" class="active">gRPC Examples</a>
    <a href="/swagger.json">Raw Spec</a>
  </div>
  <div class="container">
    <h1>POS API - gRPC Examples</h1>
    <p>Use <code>grpcurl</code> to test the gRPC API endpoints.</p>

    <div class="section">
      <h2>Health Check</h2>
      <div class="endpoint">
        <p><strong>List all services:</strong></p>
        <pre><code>grpcurl -plaintext localhost:50051 list</code></pre>
        <p><strong>Check health:</strong></p>
        <pre><code>grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check</code></pre>
      </div>
    </div>

    <div class="section">
      <h2>Authentication</h2>
      <div class="endpoint">
        <p><span class="method method-post">POST</span> <strong>Login</strong></p>
        <pre><code>grpcurl -plaintext \
  -d '{
    "username": "admin",
    "password": "admin123"
  }' \
  localhost:50051 \
  auth.v1.AuthService/Login</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-post">POST</span> <strong>Refresh Token</strong></p>
        <pre><code>grpcurl -plaintext \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }' \
  localhost:50051 \
  auth.v1.AuthService/RefreshToken</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-post">POST</span> <strong>Logout</strong></p>
        <pre><code>grpcurl -plaintext \
  -d '{
    "refresh_token": "YOUR_REFRESH_TOKEN"
  }' \
  localhost:50051 \
  auth.v1.AuthService/Logout</code></pre>
      </div>
    </div>

    <div class="section">
      <h2>Users</h2>
      <div class="endpoint">
        <p><span class="method method-get">GET</span> <strong>List Users</strong></p>
        <pre><code>grpcurl -plaintext \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "page": 1,
    "per_page": 10
  }' \
  localhost:50051 \
  user.v1.UserService/ListUsers</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-get">GET</span> <strong>Get User</strong></p>
        <pre><code>grpcurl -plaintext \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "id": 1
  }' \
  localhost:50051 \
  user.v1.UserService/GetUser</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-post">POST</span> <strong>Create User</strong></p>
        <pre><code>grpcurl -plaintext \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "username": "newcashier",
    "email": "cashier@example.com",
    "password": "password123",
    "full_name": "New Cashier",
    "role_id": 3,
    "store_id": 1
  }' \
  localhost:50051 \
  user.v1.UserService/CreateUser</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-put">PUT</span> <strong>Update User</strong></p>
        <pre><code>grpcurl -plaintext \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "id": 1,
    "email": "updated@example.com",
    "full_name": "Updated Name",
    "role_id": 2
  }' \
  localhost:50051 \
  user.v1.UserService/UpdateUser</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-delete">DELETE</span> <strong>Delete User</strong></p>
        <pre><code>grpcurl -plaintext \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -d '{
    "id": 1
  }' \
  localhost:50051 \
  user.v1.UserService/DeleteUser</code></pre>
      </div>
    </div>

    <div class="section">
      <h2>Server Reflection</h2>
      <div class="endpoint">
        <p><strong>List all services:</strong></p>
        <pre><code>grpcurl -plaintext localhost:50051 list</code></pre>
        <p><strong>List methods for a service:</strong></p>
        <pre><code>grpcurl -plaintext localhost:50051 list auth.v1.AuthService</code></pre>
        <p><strong>Describe a method:</strong></p>
        <pre><code>grpcurl -plaintext localhost:50051 describe auth.v1.AuthService.Login</code></pre>
      </div>
    </div>

    <div class="section">
      <h2>REST API (via gRPC-Gateway)</h2>
      <p>REST endpoints are available through gRPC-Gateway on port <code>8080</code>:</p>
      
      <div class="endpoint">
        <p><span class="method method-post">POST</span> <strong>/api/v1/auth/login</strong></p>
        <pre><code>curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "admin123"
  }'</code></pre>
      </div>

      <div class="endpoint">
        <p><span class="method method-get">GET</span> <strong>/api/v1/users</strong></p>
        <pre><code>curl http://localhost:8080/api/v1/users \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"</code></pre>
      </div>
    </div>
  </div>
</body>
</html>`
}

// getFilePath returns the absolute path to a file
func getFilePath(relativePath string) string {
	absPath, err := filepath.Abs(relativePath)
	if err != nil {
		return relativePath
	}
	return absPath
}
