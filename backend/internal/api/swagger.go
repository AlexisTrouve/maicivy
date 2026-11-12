package api

import (
	"embed"

	"github.com/gofiber/fiber/v2"
)

// SwaggerHandler gère les endpoints de documentation Swagger
type SwaggerHandler struct {
	swaggerFS embed.FS
}

// NewSwaggerHandler crée un nouveau handler Swagger
func NewSwaggerHandler() *SwaggerHandler {
	return &SwaggerHandler{}
}

// RegisterRoutes enregistre les routes Swagger
func (h *SwaggerHandler) RegisterRoutes(app *fiber.App) {
	// Serve OpenAPI YAML spec
	app.Get("/api/docs/openapi.yaml", h.ServeOpenAPISpec)

	// Serve Swagger UI HTML
	app.Get("/api/docs", h.ServeSwaggerUI)
	app.Get("/api/docs/", h.ServeSwaggerUI)

	// Serve Swagger UI assets (CSS, JS)
	// In production, use CDN links in HTML
}

// ServeOpenAPISpec sert le fichier OpenAPI YAML
func (h *SwaggerHandler) ServeOpenAPISpec(c *fiber.Ctx) error {
	// Path to OpenAPI spec relative to backend directory
	specPath := "docs/api/openapi.yaml"

	c.Set("Content-Type", "application/yaml")
	return c.SendFile(specPath)
}

// ServeSwaggerUI sert l'interface Swagger UI
func (h *SwaggerHandler) ServeSwaggerUI(c *fiber.Ctx) error {
	html := `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>maicivy API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui.css">
    <style>
        body {
            margin: 0;
            padding: 0;
        }
        .topbar {
            display: none;
        }
        .swagger-ui .info {
            margin: 50px 0;
        }
        .swagger-ui .info .title {
            font-size: 36px;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>

    <script src="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/api/docs/openapi.yaml',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                tryItOutEnabled: true,
                requestInterceptor: function(req) {
                    // Include credentials (cookies) in requests
                    req.credentials = 'include';
                    return req;
                },
                onComplete: function() {
                    console.log('Swagger UI loaded');
                }
            });

            window.ui = ui;
        };
    </script>
</body>
</html>
`

	c.Set("Content-Type", "text/html")
	return c.SendString(html)
}

// SetupSwaggerRoutes is a helper function to register Swagger routes on the app
func SetupSwaggerRoutes(app *fiber.App) {
	handler := NewSwaggerHandler()
	handler.RegisterRoutes(app)
}
