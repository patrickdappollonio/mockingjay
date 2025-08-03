# Mockingjay

A pluggable HTTP server configurable via YAML that uses Go templates to render dynamic responses based on request data.

## Overview

Mockingjay is a lightweight, flexible HTTP server designed for:
- **API mocking and prototyping**: Quickly create mock endpoints for development and testing
- **Dynamic response generation**: Use Go templates with rich context data from incoming requests
- **Configuration-driven routing**: Define routes, headers, and responses in simple YAML files
- **Template-based responses**: Leverage the power of Go's `html/template` with 100+ helper functions

### Key Features

- **Static and regex-based routing** with named capture groups
- **Inline or file-based templates** for maximum flexibility
- **Rich template context** including headers, query params, JSON body, and URL parameters
- **100+ template helper functions** from [Masterminds/sprig](https://github.com/Masterminds/sprig)
- **Header matching** with literal strings and regex patterns
- **Custom response headers** with template support
- **Request/response middleware** with CORS and logging support
- **Built-in health check endpoint** with server metrics
- **Configuration validation** command-line tool
- **Hot-reload** configuration changes without restart
- **Structured logging** with `log/slog`
- **Graceful shutdown** with signal handling

## Installation

### From Source

```bash
go install github.com/patrickdappollonio/mockingjay@latest
```

### Build from Repository

```bash
git clone https://github.com/patrickdappollonio/mockingjay.git
cd mockingjay
go build -o mockingjay .
```

## Quick Start

There are many examples in the [`examples` directory](examples), feel free to use them as a starting point, or alternatively, try this configuration:

1. **Create a configuration file** (`config.yaml`):

```yaml
routes:
  - path: "/hello"
    verb: "GET"
    template: |
      Hello, World! 🌍
      Current time: {{ now | date "2006-01-02 15:04:05" }}

  - path: "/^/user/(?P<name>[^/]+)$/"
    verb: "GET"
    template: |
      Hello, {{ .Params.name }}! 👋
      You're using: {{ .Headers.User-Agent }}
```

2. **Start the server**:

```bash
mockingjay -config config.yaml -port 8080
```

3. **Test your endpoints**:

```bash
curl http://localhost:8080/hello
curl http://localhost:8080/user/alice
```

## Command Line Options

```bash
mockingjay [options]

Options:
  -config string    Path to configuration file (default "config.yaml")
  -port string      Server port (default "8080")
  -debug           Enable debug logging (default false)
  -validate        Validate configuration file and exit (default false)
```

### Examples

```bash
# Basic usage
mockingjay

# Custom config file and port
mockingjay -config my-routes.yaml -port 3000

# Enable debug logging
mockingjay -config config.yaml -debug

# Validate configuration without starting server
mockingjay -config config.yaml -validate
```

## Configuration Reference

### Basic Structure

```yaml
# Optional: Middleware configuration
middleware:
  enabled:
    - type: "cors"                  # CORS middleware
      config:
        allow_origins: ["*"]
        allow_methods: ["GET", "POST", "PUT", "DELETE"]
    - type: "basicauth"             # Basic authentication middleware
      config:
        username: "admin"
        password: "secret"
        paths:
          include: ["/admin"]
    - type: "logger"                # Request logging middleware
      config:
        format: "text"
        skip_paths: ["/health"]

routes:
  - path: "/api/endpoint"           # Required: URL path (literal or regex)
    verb: "GET"                     # Optional: HTTP method (default: any)
    template: "Hello World"         # Either template (inline)
    # OR
    template_file: "./hello.tmpl"   # OR template_file (external file)
    matchHeaders:                   # Optional: Required request headers
      Authorization: "Bearer *"
      Content-Type: "application/json"
    responseHeaders:                # Optional: Custom response headers
      Content-Type: "application/json"
      X-Server: "mockingjay"
```

### Path Patterns

#### Literal Paths
```yaml
- path: "/api/users"              # Exact match
- path: "/healthz"                # Exact match
```

#### Regex Paths (wrapped in `/...$/`)
```yaml
- path: "/^/user/(?P<id>\\d+)$/"             # User with numeric ID
- path: "/^/api/(?P<version>v\\d+)/users$/"  # API versioning
- path: "/^/files/(?P<path>.+)$/"            # Capture file paths
```

I recommend you test your regex paths with [`regex101.com`](https://regex101.com/).

**Named Capture Groups**: Use `(?P<name>pattern)` to capture URL parameters accessible as `{{ .Params.name }}` in templates.

### HTTP Verbs

```yaml
verb: "GET"     # GET requests only
verb: "POST"    # POST requests only
verb: "PUT"     # PUT requests only
verb: "DELETE"  # DELETE requests only
# Omit verb to match any HTTP method
```

### Header Matching

Match requests based on headers (case-insensitive header names):

```yaml
matchHeaders:
  # Exact string match
  Authorization: "Bearer secret-token"

  # Regex pattern (wrapped in /.../)
  Authorization: "/Bearer .+/"
  Content-Type: "/application\\/(json|xml)/"

  # Multiple headers (ALL must match)
  X-API-Key: "12345"
  Content-Type: "application/json"
```

### Custom Response Headers

Set custom headers on responses (supports template syntax):

```yaml
responseHeaders:
  # Static headers
  Content-Type: "application/json"
  X-Powered-By: "mockingjay"

  # Dynamic headers using template context
  X-Request-ID: "{{ .Headers.X-Request-ID }}"
  X-User-Agent: "{{ .Headers.User-Agent }}"
  X-Timestamp: "{{ now | date \"2006-01-02T15:04:05Z07:00\" }}"
```

## Middleware

Mockingjay supports configurable middleware for request/response processing. Middleware is executed in the order defined in the configuration.

### Middleware Configuration

```yaml
middleware:
  enabled:
    - type: "cors"
      config:
        # CORS configuration
    - type: "logger"
      config:
        # Logger configuration
```

### CORS Middleware

Enable Cross-Origin Resource Sharing (CORS) support:

```yaml
middleware:
  enabled:
    - type: "cors"
      config:
        allow_origins: ["http://localhost:3000", "https://example.com"]  # Allowed origins
        allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]       # Allowed HTTP methods
        allow_headers: ["Content-Type", "Authorization", "X-API-Key"]    # Allowed headers
        expose_headers: ["X-Total-Count"]                                # Exposed headers
        allow_credentials: true                                          # Allow credentials
        max_age: 3600                                                    # Preflight cache time (seconds)
```

#### CORS Configuration Options

| Option              | Type       | Default                                       | Description                             |
| ------------------- | ---------- | --------------------------------------------- | --------------------------------------- |
| `allow_origins`     | `[]string` | `["*"]`                                       | Allowed origins                         |
| `allow_methods`     | `[]string` | `["GET", "POST", "PUT", "DELETE", "OPTIONS"]` | Allowed HTTP methods                    |
| `allow_headers`     | `[]string` | `["Content-Type", "Authorization"]`           | Allowed request headers                 |
| `expose_headers`    | `[]string` | `[]`                                          | Headers exposed to the client           |
| `allow_credentials` | `bool`     | `false`                                       | Allow credentials in CORS requests      |
| `max_age`           | `int`      | `3600`                                        | Preflight response cache time (seconds) |

#### CORS Examples

**Allow all origins:**
```yaml
middleware:
  enabled:
    - type: "cors"
      config:
        allow_origins: ["*"]
```

**Specific origins with credentials:**
```yaml
middleware:
  enabled:
    - type: "cors"
      config:
        allow_origins: ["https://app.example.com", "https://admin.example.com"]
        allow_credentials: true
        allow_headers: ["Content-Type", "Authorization", "X-API-Key"]
```

### Logger Middleware

Enhanced request logging with configurable options:

```yaml
middleware:
  enabled:
    - type: "logger"
      config:
        format: "text"                    # "text" or "json"
        level: "info"                     # "debug", "info", "warn", "error"
        skip_paths: ["/health", "/ping"]  # Paths to skip logging
```

### Basic Auth Middleware

HTTP Basic Authentication with flexible path matching:

```yaml
middleware:
  enabled:
    - type: "basicauth"
      config:
        username: "admin"                 # Required: Username for authentication
        password: "secret123"             # Required: Password for authentication
        realm: "Restricted Area"          # Optional: Authentication realm
        paths:                            # Optional: Path matching rules
          include:                        # Paths that require authentication
            - "/admin"                    # Literal path
            - "/^/api/v\\d+/private$/"    # Regex pattern
          exclude:                        # Paths to exclude from authentication
            - "/admin/health"             # Health checks
            - "/api/v1/public"            # Public endpoints
```

#### Basic Auth Configuration Options

| Option     | Type             | Default             | Description                                            |
| ---------- | ---------------- | ------------------- | ------------------------------------------------------ |
| `username` | `string`         | *required*          | Username for authentication                            |
| `password` | `string`         | *required*          | Password for authentication                            |
| `realm`    | `string`         | `"Restricted Area"` | Authentication realm displayed in browser login prompt |
| `paths`    | `BasicAuthPaths` | `{}`                | Path matching configuration (include/exclude)          |

**Path Matching Configuration:**

| Option    | Type       | Default | Description                                                          |
| --------- | ---------- | ------- | -------------------------------------------------------------------- |
| `include` | `[]string` | `[]`    | Paths requiring authentication (literal or regex)                    |
| `exclude` | `[]string` | `[]`    | Paths to exclude from authentication (takes precedence over include) |

**Path Matching Rules:**
- **No include patterns**: Authentication required for all paths (except excluded)
- **With include patterns**: Authentication only required for matching paths
- **Exclude patterns**: Always take precedence over include patterns
- **Regex patterns**: Wrap in `/^...$/` (same as routes)
- **Literal paths**: Exact string match

#### Basic Auth Examples

**Protect all admin paths:**
```yaml
middleware:
  enabled:
    - type: "basicauth"
      config:
        username: "admin"
        password: "secret123"
        paths:
          include: ["/^/admin/.*$/"]
```

**Protect specific API versions:**
```yaml
middleware:
  enabled:
    - type: "basicauth"
      config:
        username: "apiuser"
        password: "apikey123"
        realm: "API Access"
        paths:
          include:
            - "/^/api/v[12]/private$/"
            - "/admin"
          exclude:
            - "/admin/health"
```

**Protect everything except public endpoints:**
```yaml
middleware:
  enabled:
    - type: "basicauth"
      config:
        username: "user"
        password: "pass"
        # No include patterns = protect everything
        paths:
          exclude:
            - "/public"
            - "/health"
            - "/^/api/v\\d+/public$/"
```

#### Logger Configuration Options

| Option       | Type       | Default  | Description                                 |
| ------------ | ---------- | -------- | ------------------------------------------- |
| `format`     | `string`   | `"text"` | Log format: "text" or "json"                |
| `level`      | `string`   | `"info"` | Log level: "debug", "info", "warn", "error" |
| `skip_paths` | `[]string` | `[]`     | Request paths to skip from logging          |

#### Logger Examples

**Basic logging:**
```yaml
middleware:
  enabled:
    - type: "logger"
```

**Skip health check endpoints:**
```yaml
middleware:
  enabled:
    - type: "logger"
      config:
        skip_paths: ["/health", "/healthz", "/ping"]
```

### Complete Middleware Example

```yaml
middleware:
  enabled:
    # Enable CORS for frontend applications
    - type: "cors"
      config:
        allow_origins: ["http://localhost:3000", "https://myapp.com"]
        allow_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
        allow_headers: ["Content-Type", "Authorization", "X-Requested-With"]
        allow_credentials: true
        max_age: 3600

    # Enable basic authentication for admin endpoints
    - type: "basicauth"
      config:
        username: "admin"
        password: "secure123"
        realm: "Admin Area"
        paths:
          include: ["/^/admin/.*$/", "/api/private"]
          exclude: ["/admin/health"]

    # Enable request logging (skip health checks)
    - type: "logger"
      config:
        format: "text"
        level: "info"
        skip_paths: ["/health", "/admin/health"]

routes:
  # Public API endpoint
  - path: "/api/data"
    verb: "GET"
    template: |
      {
        "message": "Public CORS-enabled API endpoint",
        "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}"
      }
    responseHeaders:
      Content-Type: "application/json"

  # Protected admin endpoint
  - path: "/admin/dashboard"
    verb: "GET"
    template: |
      {
        "message": "Protected admin dashboard",
        "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}",
        "authenticated": true
      }
    responseHeaders:
      Content-Type: "application/json"
```

## Built-in Health Check

Mockingjay includes a built-in health check endpoint at `/health` that provides server status and metrics:

```bash
curl http://localhost:8080/health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "dev",
  "timestamp": "2025-08-03T01:59:34.113901-04:00",
  "uptime": "5.362699416s",
  "routes": 3,
  "config_file": "config.yaml",
  "go_version": "go1.24.0",
  "memory": {
    "alloc_bytes": 596272,
    "heap_alloc_bytes": 596272,
    "sys_bytes": 7228432,
    "total_alloc_bytes": 596272
  }
}
```

The health check endpoint:
- **Always available** regardless of configuration
- **GET requests only** (other methods return 404)
- **Thread-safe** during config reloads
- **JSON response** with comprehensive server information

## Template Syntax

Mockingjay uses Go's [`html/template`](https://pkg.go.dev/html/template) engine with automatic HTML escaping.

### Template Context

Every template has access to:

```go
{
  "Request": *http.Request,              // Raw HTTP request object
  "Headers": map[string]string,          // Request headers (case-insensitive keys)
  "Query":   map[string]string,          // Query parameters
  "Body":    interface{},                // Parsed JSON body (if applicable)
  "Params":  map[string]string           // URL parameters from regex captures
}
```

### Basic Template Examples

```yaml
template: |
  Request Method: {{ .Request.Method }}
  Request Path: {{ .Request.URL.Path }}
  User Agent: {{ .Headers.User-Agent }}
  Query Param 'debug': {{ .Query.debug }}
  Captured 'id' parameter: {{ .Params.id }}
```

### JSON Body Access

For requests with `Content-Type: application/json`:

```yaml
template: |
  Hello {{ .Body.name }}!
  Your email is: {{ .Body.email }}

  Full JSON body: {{ .Body | toPrettyJson }}
```

## 🧰 Template Helper Functions

Mockingjay includes **100+ helper functions** from [Masterminds/sprig](http://masterminds.github.io/sprig/) plus custom functions:

### Custom Functions

| Function                         | Description                | Example                                    |
| -------------------------------- | -------------------------- | ------------------------------------------ |
| `header "X-API-Key"`             | Get request header value   | `{{ header "Authorization" }}`             |
| `query "page"`                   | Get query parameter value  | `{{ query "limit" }}`                      |
| `jsonBody`                       | Parse request body as JSON | `{{ jsonBody.user.name }}`                 |
| `trimPrefix "/api" "/api/users"` | Remove prefix from string  | `{{ trimPrefix "/v1" .Request.URL.Path }}` |

### Popular Sprig Functions

| Category         | Functions                                     | Examples                              |
| ---------------- | --------------------------------------------- | ------------------------------------- |
| **Strings**      | `upper`, `lower`, `title`, `trim`, `contains` | `{{ .Params.name \| upper }}`         |
| **Numbers**      | `add`, `sub`, `mul`, `div`, `mod`             | `{{ add 1 2 }}`                       |
| **Dates**        | `now`, `date`, `dateModify`                   | `{{ now \| date "2006-01-02" }}`      |
| **Arrays**       | `slice`, `first`, `last`, `join`              | `{{ .Headers \| keys \| join ", " }}` |
| **Conditionals** | `if`, `eq`, `ne`, `lt`, `gt`                  | `{{ if eq .Request.Method "POST" }}`  |
| **JSON**         | `toJson`, `toPrettyJson`, `fromJson`          | `{{ .Body \| toPrettyJson }}`         |

See the [complete Sprig documentation](http://masterminds.github.io/sprig/) for all available functions.

### Template Examples

#### Simple JSON Response
```yaml
template: |
  {
    "message": "Hello {{ .Params.name }}",
    "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}",
    "method": "{{ .Request.Method }}"
  }
```

#### HTML Response
```yaml
template: |
  <!DOCTYPE html>
  <html>
  <head><title>User Profile</title></head>
  <body>
    <h1>Welcome, {{ .Params.name | title }}!</h1>
    <p>Request from: {{ .Headers.User-Agent }}</p>
    <p>Time: {{ now | date "Monday, January 2, 2006" }}</p>
  </body>
  </html>
```

#### Conditional Logic
```yaml
template: |
  {{ if eq .Request.Method "POST" -}}
  Created user: {{ .Body.name }}
  {{ else -}}
  Retrieved user: {{ .Params.id }}
  {{ end -}}

  {{ if .Query.debug -}}
  Debug mode: ON
  {{ end -}}
```

## Troubleshooting

### Debug Mode

Enable debug logging to see detailed request information:

```bash
mockingjay -config config.yaml -debug
```

Debug output includes:
- Request details (method, path, headers)
- Route matching process
- Template execution context
- Response generation

### Hot-Reload Support

Mockingjay supports hot-reloading of configuration files:
- **File watching**: Automatically detects changes to the config file
- **Atomic reloads**: Routes and middleware are updated atomically
- **Zero downtime**: Server continues serving requests during reload
- **Error handling**: Invalid configurations don't affect running server
- **Thread-safe**: Safe concurrent access during reloads
