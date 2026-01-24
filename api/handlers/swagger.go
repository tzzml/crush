package handlers

import (
	"context"
	"net/http"

	hertzapp "github.com/cloudwego/hertz/pkg/app"
	"github.com/swaggo/swag"
	_ "github.com/charmbracelet/crush/docs"
)

// SwaggerHTML 是 Swagger UI 的 HTML 页面
const SwaggerHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API - Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        #swagger-ui { max-width: 1460px; margin: 0 auto; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger/doc.json',
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
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 2,
                docExpansion: "list",
                persistAuthorization: true,
                displayRequestDuration: true
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`

// HandleSwaggerUI 处理 GET /swagger 请求
func HandleSwaggerUI(c context.Context, ctx *hertzapp.RequestContext) {
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.Response.SetBody([]byte(SwaggerHTML))
}

// HandleSwaggerJSON 处理 GET /swagger/doc.json 请求
func HandleSwaggerJSON(c context.Context, ctx *hertzapp.RequestContext) {
	doc, err := swag.ReadDoc("swagger")
	if err != nil {
		ctx.JSON(500, map[string]string{"error": "failed to read swagger doc"})
		return
	}
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json; charset=utf-8")
	ctx.Response.SetBody([]byte(doc))
}

// HandleIndexRedirect 处理 GET / 请求
func HandleIndexRedirect(c context.Context, ctx *hertzapp.RequestContext) {
	ctx.Redirect(http.StatusFound, []byte("/swagger"))
}
