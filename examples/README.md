# Mockingjay Examples

This directory contains carefully curated examples showcasing different aspects of mockingjay. Each example focuses on specific features to help you learn incrementally.

## Quick Start

Run any example with:
```bash
mockingjay -config examples/{example-name}.yaml
```

## Example Categories

### 🚀 Getting Started
- **[hello-world.yaml](hello-world.yaml)** - Simple routes with static responses and URL parameters
- **[json-echo.yaml](json-echo.yaml)** - JSON request/response handling with dynamic content
- **[fake-data-demo.yaml](fake-data-demo.yaml)** - Full set of fake data generation examples

### 🛡️ Advanced Configuration
- **[advanced-middleware.yaml](advanced-middleware.yaml)** - Production-ready middleware (CORS, auth, logging)
- **[custom-timeouts.yaml](custom-timeouts.yaml)** - Server and middleware timeout configuration
- **[response-headers.yaml](response-headers.yaml)** - Static and dynamic response header manipulation

### 📄 Templates & Files
- **[template-files.yaml](template-files.yaml)** - Using external template files instead of inline templates

### 📋 Reference & Documentation
- **[all-options-reference.yaml](all-options-reference.yaml)** - Complete configuration reference with all options, defaults, and comments

### 🎯 Complete Demo
- **[full-featured.yaml](full-featured.yaml)** - Showcase of all major features

## Custom Configuration Examples

If you prefer to start with inline configuration examples:

### 1. Simple API Mock

This example demonstrates how to create a simple API mock that returns a JSON response.

Store the following as the configuration file. This configuration will:
- Enable CORS for all origins
- Log all requests to the console
- Skip logging the health check endpoint
- Return a JSON response with the status and timestamp

```yaml
middleware:
  enabled:
    - type: "cors"
      config:
        allow_origins: ["*"]
    - type: "logger"
      config:
        skip_paths: ["/health"]

routes:
  # Health check
  - path: "/health"
    method: "GET"
    template: |
      {
        "status": "ok",
        "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}"
      }
    response_headers:
      Content-Type: "application/json"

  # User CRUD operations
  - path: "/^/users/(?P<id>\\d+)$/"
    method: "GET"
    template: |
      {
        "id": {{ .Params.id }},
        "name": "User {{ .Params.id }}",
        "email": "user{{ .Params.id }}@example.com"
      }
    response_headers:
      Content-Type: "application/json"

  - path: "/users"
    method: "POST"
    template: |
      {
        "id": {{ randInt 1000 9999 }},
        "name": "{{ .Body.name }}",
        "email": "{{ .Body.email }}",
        "created_at": "{{ now | date "2006-01-02T15:04:05Z07:00" }}"
      }
    response_headers:
      Content-Type: "application/json"
```

Run the following command to start the server, replacing `/path/to/config.yaml` with the path to the configuration file you saved above.

```bash
mockingjay -config /path/to/config.yaml
```

### 2. Header-based Routing

This example demonstrate how to route based on headers.

Store the following as the configuration file. This configuration will:
- Require an API key in the Authorization header
- Return a JSON response with the status and timestamp

```yaml
routes:
  # Require API key
  - path: "/api/secure"
    match_headers:
      Authorization: "/Bearer .+/"
    template: |
      {
        "message": "Access granted",
        "token": "{{ trimPrefix "Bearer " .Headers.Authorization }}"
      }
    response_headers:
      Content-Type: "application/json"

  # Version-specific endpoint
  - path: "/api/data"
    match_headers:
      Accept: "/application\\/vnd\\.api\\+json;version=([12])/"
    template: |
      API Version: {{ regexFind "version=([12])" .Headers.Accept }}
      Data: {{ .Body | toPrettyJson }}
```

### 3. HTML Templates with Styling

This example demonstrates how to create HTML responses with inline templates.

This configuration will:
- Return a styled HTML response with the user's profile
- Populate the page with information from the request
- Use CSS styling for a professional look

`config.yaml`:
```yaml
routes:
  - path: "/^/profile/(?P<name>[^/]+)$/"
    method: "GET"
    template: |
      <!DOCTYPE html>
      <html lang="en">
      <head>
          <meta charset="UTF-8">
          <meta name="viewport" content="width=device-width, initial-scale=1.0">
          <title>{{ .Params.name | title }}'s Profile</title>
          <style>
              body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                     max-width: 800px; margin: 40px auto; padding: 20px; background: #f5f7fa; }
              .profile-card { background: white; border-radius: 12px; padding: 30px;
                             box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1); }
              h1 { color: #2c3e50; margin-bottom: 20px; }
              .info { background: #ecf0f1; padding: 15px; border-radius: 8px; margin-top: 20px; }
              .info ul { margin: 0; padding-left: 20px; }
              .info li { margin: 5px 0; }
          </style>
      </head>
      <body>
          <div class="profile-card">
              <h1>Welcome, {{ .Params.name | title }}!</h1>
              <p>This is your profile page generated dynamically from the request.</p>

              <div class="info">
                  <h3>Request Information</h3>
                  <ul>
                      <li><strong>Method:</strong> {{ .Request.Method }}</li>
                      <li><strong>Path:</strong> {{ .Request.URL.Path }}</li>
                      <li><strong>User Agent:</strong> {{ .Headers.Get "User-Agent" | trunc 50 }}...</li>
                      <li><strong>Timestamp:</strong> {{ now | date "2006-01-02 15:04:05" }}</li>
                  </ul>
              </div>
          </div>
      </body>
      </html>
    response_headers:
      Content-Type: "text/html; charset=utf-8"
```

---

## About This Collection

These examples have been carefully selected to provide coverage of mockingjay's features. Each example builds upon concepts from the previous ones, making it easy to learn incrementally.

**Learning Path Recommendation:**
1. Start with `hello-world.yaml` for basic concepts
2. Move to `json-echo.yaml` for JSON handling
3. Try `fake-data-demo.yaml` for generating realistic test data
4. Explore `advanced-middleware.yaml` for production patterns
5. Try `template-files.yaml` for external templates
6. Experiment with `response-headers.yaml` for header manipulation
7. Configure `custom-timeouts.yaml` for performance tuning
8. Review `full-featured.yaml` to see everything combined
9. Try `custom-delimiters.yaml` for custom template delimiters

**For Reference:**
- Keep `all-options-reference.yaml` handy as your complete configuration guide - copy it, delete what you don't need, and customize the rest!
