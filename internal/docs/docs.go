package docs

import _ "embed"

//go:embed openapi.yaml
var OpenAPIYAML string

const DocsHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>RentCar API Docs</title>
  <style>
    body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 0; color: #0f172a; background: #f8fafc; }
    main { max-width: 900px; margin: 0 auto; padding: 48px 20px; }
    a { color: #2563eb; font-weight: 700; }
    pre { overflow: auto; background: #0f172a; color: #e2e8f0; padding: 20px; border-radius: 8px; }
  </style>
</head>
<body>
  <main>
    <h1>RentCar API Docs</h1>
    <p>OpenAPI schema: <a href="/openapi.yaml">/openapi.yaml</a></p>
    <p>Import this YAML into Postman, Insomnia, Swagger Editor, or Scalar to explore the API.</p>
    <pre>curl http://localhost:8080/openapi.yaml</pre>
  </main>
</body>
</html>`
