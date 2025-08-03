# Mockingjay Examples

This directory contains examples of how to use `mockingjay` with different configurations. Navigate the directory and run each example to try it out:

```bash
mockingjay -config examples/json-echo.yaml
```

The examples range from simple, to complex, to using middlewares, custom templates, pattern matching, and more.

## Custom examples

If you prefer to have a quick start, you can try the following examples:

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
    verb: "GET"
    template: |
      {
        "status": "ok",
        "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}"
      }
    responseHeaders:
      Content-Type: "application/json"

  # User CRUD operations
  - path: "/^/users/(?P<id>\\d+)$/"
    verb: "GET"
    template: |
      {
        "id": {{ .Params.id }},
        "name": "User {{ .Params.id }}",
        "email": "user{{ .Params.id }}@example.com"
      }
    responseHeaders:
      Content-Type: "application/json"

  - path: "/users"
    verb: "POST"
    template: |
      {
        "id": {{ randInt 1000 9999 }},
        "name": "{{ .Body.name }}",
        "email": "{{ .Body.email }}",
        "created_at": "{{ now | date "2006-01-02T15:04:05Z07:00" }}"
      }
    responseHeaders:
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
    matchHeaders:
      Authorization: "/Bearer .+/"
    template: |
      {
        "message": "Access granted",
        "token": "{{ trimPrefix "Bearer " .Headers.Authorization }}"
      }
    responseHeaders:
      Content-Type: "application/json"

  # Version-specific endpoint
  - path: "/api/data"
    matchHeaders:
      Accept: "/application\\/vnd\\.api\\+json;version=([12])/"
    template: |
      API Version: {{ regexFind "version=([12])" .Headers.Accept }}
      Data: {{ .Body | toPrettyJson }}
```

### 3. File-based Templates

This example demonstrates how to use file-based templates.

Store the following as the configuration file. This configuration will:
- Return a HTML response with the user's profile, populated with information from the request
- Use the `templates/user-profile.html` template

`templates/user-profile.html`:
```html
<!DOCTYPE html>
<html>
<head>
    <title>{{ .Params.name | title }}'s Profile</title>
</head>
<body>
    <h1>Welcome, {{ .Params.name | title }}!</h1>
    <div>
        <h2>Request Info</h2>
        <ul>
            <li>Method: {{ .Request.Method }}</li>
            <li>Path: {{ .Request.URL.Path }}</li>
            <li>User Agent: {{ .Headers.User-Agent }}</li>
        </ul>
    </div>
</body>
</html>
```

`config.yaml`:
```yaml
routes:
  - path: "/^/profile/(?P<name>[^/]+)$/"
    verb: "GET"
    template_file: "./templates/user-profile.html"
    responseHeaders:
      Content-Type: "text/html; charset=utf-8"
```
