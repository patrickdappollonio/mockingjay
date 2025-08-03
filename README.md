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
- **Inline or file-based templates** for maximum flexibility with pre-compilation for performance
- **Rich template context** including headers, query params, JSON body, and URL parameters
- **100+ template helper functions** from [Masterminds/sprig](https://github.com/Masterminds/sprig) plus 80+ functions that generate fake data
- **Header matching** with literal strings and regex patterns
- **Custom response headers** with template support
- **Request/response middleware** with CORS, authentication, and logging support
- **Request timeout handling** with configurable server and middleware timeouts
- **Built-in health check endpoint** with server metrics
- **Configuration validation** with template compilation checking
- **Hot-reload** configuration changes without restart
- **Structured logging** with `log/slog`
- **Graceful shutdown** with signal handling

## Installation

### Homebrew for macOS and Linux

The easiest way to get started with Mockingjay is to install it using Homebrew. This is available on M1+ on MacOS and x86_64 and ARM64 on Linux.

```bash
brew install patrickdappollonio/tap/mockingjay
```

### Binaries

You can download the latest binary from the [releases page](https://github.com/patrickdappollonio/mockingjay/releases).

### Docker

You can run Mockingjay in a container using the `docker` command. We provide `latest` which always points to the newest version, and `v1` which points to the latest stable version. We recommend using `v1` for production.

```bash
docker pull ghcr.io/patrickdappollonio/mockingjay:v1

# or using bleeding edge:
# docker pull ghcr.io/patrickdappollonio/mockingjay:latest
```

## Quick Start

There are many examples in the [`examples` directory](examples), feel free to use them as a starting point. For a complete configuration reference with all options and defaults, check out [`examples/all-options-reference.yaml`](examples/all-options-reference.yaml) - copy it, delete what you don't need, and customize!

Alternatively, try this basic configuration:

1. **Create a configuration file** (`config.yaml`):

```yaml
routes:
  - path: "/hello"
    method: "GET"
    template: |
      Hello, World! 🌍
      Current time: {{ now | date "2006-01-02 15:04:05" }}

  - path: "/^/user/(?P<name>[^/]+)$/"
    method: "GET"
    template: |
      Hello, {{ .Params.name }}! 👋
      You're using: {{ .Headers.User-Agent }}

  - path: "/fake-user"
    method: "GET"
    template: |
      {
        "id": "{{ fakeUUID }}",
        "name": "{{ fakeName }}",
        "email": "{{ fakeEmail }}",
        "company": "{{ fakeCompany }}"
      }
```

2. **Start the server**:

```bash
mockingjay --config config.yaml --port 8080
```

3. **Test your endpoints**:

```bash
curl http://localhost:8080/hello
curl http://localhost:8080/user/alice
curl http://localhost:8080/fake-user
```

## Command Line Options

```bash
mockingjay [flags]

Flags:
  -c, --config string   path to configuration file (default "config.yaml")
  -p, --port string     server port (default "8080")
  -d, --debug           enable debug logging
      --validate        validate configuration file and exit
  -v, --version         version for mockingjay
  -h, --help            help for mockingjay
```

### Examples

```bash
# Basic usage using the default config file and port
mockingjay

# Custom config file and port (long flags)
mockingjay --config my-routes.yaml --port 3000

# Custom config file and port (short flags)
mockingjay -c my-routes.yaml -p 3000

# Enable debug logging
mockingjay --config config.yaml --debug

# Validate configuration without starting server
mockingjay --config config.yaml --validate
```

## Configuration Validation

Mockingjay provides a validation command that checks your configuration for errors before starting the server.

### Using the Validate Command

```bash
# Validate the default config.yaml file
mockingjay --validate

# Validate a specific configuration file
mockingjay --validate --config my-config.yaml

# Using short flags
mockingjay --validate -c my-config.yaml

# Validate with debug output for detailed error information
mockingjay --validate --config config.yaml --debug
```

### What Gets Validated

The validation process performs checks on:

- **YAML Syntax**: Ensures the configuration file is valid YAML
- **Configuration Structure**: Validates all required fields and data types
- **Route Configuration**: Checks paths, HTTP methods, and route definitions
- **Template Compilation**: Compiles all templates (inline and file-based) to catch syntax errors
- **Response Header Templates**: Validates custom response header template syntax
- **Regex Patterns**: Validates regex syntax in path patterns and header matching
- **File Access**: Verifies that template files exist and are readable
- **Header Validation**: Checks HTTP header name validity and regex patterns

### Validation Output

**Successful Validation:**
```bash
$ mockingjay --validate --config examples/hello-world.yaml
✅ Configuration file "examples/hello-world.yaml" is valid
   - Found 3 routes
   - All templates compiled successfully
   - All validation checks passed
```

**Failed Validation:**
```bash
$ mockingjay --validate --config broken-config.yaml
❌ Configuration validation failed:
   route[0] template compilation failed: template compilation error in inline:
   failed to parse template: template: validation_route_0_GET_test:1:
   function "invalidFunction" not defined
```

## Configuration Reference

### Basic Structure

```yaml
# Optional: Server configuration
server:
  timeouts:
    read: "15s"              # Maximum time to read entire request
    write: "15s"             # Maximum time to write response
    idle: "60s"              # Maximum time to wait for next request on keep-alive
    read_header: "5s"        # Maximum time to read request headers
    request: "30s"           # Per-request timeout (applied via middleware)
    shutdown: "30s"          # Graceful shutdown timeout

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
    - type: "timeout"               # Request timeout monitoring middleware
      config:
        duration: "30s"
    - type: "logger"                # Request logging middleware
      config:
        format: "text"
        skip_paths: ["/health"]

routes:
  - path: "/api/endpoint"           # Required: URL path (literal or regex)
    method: "GET"                     # Optional: HTTP method (default: any)
    template: "Hello World"         # Either template (inline)
    # OR
    template_file: "./hello.tmpl"   # OR template_file (external file)
    match_headers:                  # Optional: Required request headers
      Authorization: "Bearer *"
      Content-Type: "application/json"
    response_headers:               # Optional: Custom response headers
      Content-Type: "application/json"
      X-Server: "mockingjay"
```

### Path Patterns

#### Literal Paths
```yaml
- path: "/api/users"              # Exact match
- path: "/healthz"                # Exact match
```

#### Regex Paths (wrapped in `/.../`)
```yaml
- path: "/^/user/(?P<id>\\d+)$/"             # User with numeric ID
- path: "/^/api/(?P<version>v\\d+)/users$/"  # API versioning
- path: "/^/files/(?P<path>.+)$/"            # Capture file paths
```

I recommend you test your regex paths with [`regex101.com`](https://regex101.com/).

**Named Capture Groups**: Use `(?P<name>pattern)` to capture URL parameters accessible as `{{ .Params.name }}` in templates.

### Server Configuration

Configure server-level settings including timeout handling:

```yaml
server:
  timeouts:
    read: "15s"              # Maximum time to read entire request (default: 15s)
    write: "15s"             # Maximum time to write response (default: 15s)
    idle: "60s"              # Maximum time to wait for next request on keep-alive (default: 60s)
    read_header: "5s"        # Maximum time to read request headers (default: 5s)
    request: "30s"           # Per-request timeout for middleware monitoring (default: 30s)
    shutdown: "30s"          # Graceful shutdown timeout (default: 30s)
```

#### Timeout Configuration Options

| Option        | Type       | Default | Description                                             |
| ------------- | ---------- | ------- | ------------------------------------------------------- |
| `read`        | `duration` | `"15s"` | Maximum time to read the entire request body            |
| `write`       | `duration` | `"15s"` | Maximum time to write the response                      |
| `idle`        | `duration` | `"60s"` | Maximum time to wait for next request on keep-alive     |
| `read_header` | `duration` | `"5s"`  | Maximum time to read request headers                    |
| `request`     | `duration` | `"30s"` | Server-level request monitoring timeout (logs warnings) |
| `shutdown`    | `duration` | `"30s"` | Maximum time to wait for graceful shutdown              |

**Duration Format**: Use Go duration strings like `"30s"`, `"5m"`, `"1h30m"`.

#### Server Configuration Examples

**Basic timeout configuration:**
```yaml
server:
  timeouts:
    read: "15s"
    write: "20s"
    request: "45s"
```

**Extended timeouts for file uploads:**
```yaml
server:
  timeouts:
    read: "60s"        # Allow longer read for large uploads
    write: "30s"       # Standard write timeout
    idle: "120s"       # Extended idle for long polling
    request: "120s"    # Monitor long-running requests
    shutdown: "30s"    # Allow time for cleanup
```

### HTTP Methods

```yaml
method: "GET"     # GET requests only
method: "POST"    # POST requests only
method: "PUT"     # PUT requests only
method: "DELETE"  # DELETE requests only
# Omit method to match any HTTP method
```

### Header Matching

Match requests based on headers (case-insensitive header names):

```yaml
match_headers:
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
response_headers:
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
    - type: "basicauth"
      config:
        # Basic authentication configuration
    - type: "timeout"
      config:
        # Timeout monitoring configuration
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

### Timeout Middleware

Enforce request timeouts by cancelling requests that exceed configured duration and returning `408 Request Timeout`:

```yaml
middleware:
  enabled:
    - type: "timeout"
      config:
        duration: "30s"                   # Request timeout duration (default: 30s)
```

#### Timeout Middleware Configuration Options

| Option     | Type       | Default | Description                                  |
| ---------- | ---------- | ------- | -------------------------------------------- |
| `duration` | `duration` | `"30s"` | Maximum request duration before cancellation |

#### How Timeout Middleware Works

The timeout middleware provides **request-level timeout enforcement**:

1. **Context Cancellation**: Creates a timeout context for each request
2. **Template Buffering**: Templates are rendered to a buffer with timeout protection
3. **Request Termination**: Returns `408 Request Timeout` if the timeout is exceeded
4. **Immediate Response**: Clients receive timeout response without waiting for completion
5. **Structured Logging**: Logs timeout events with detailed timing information

#### Timeout Middleware Examples

**Basic timeout enforcement:**
```yaml
middleware:
  enabled:
    - type: "timeout"
      config:
        duration: "60s"  # Cancel requests taking longer than 60 seconds
```

**Quick timeout for fast APIs:**
```yaml
middleware:
  enabled:
    - type: "timeout"
      config:
        duration: "5s"   # Cancel requests taking longer than 5 seconds
```

**Combined with server timeouts:**
```yaml
server:
  timeouts:
    read: "30s"         # Network-level read timeout
    write: "30s"        # Network-level write timeout
    request: "45s"      # Server-level monitoring timeout

middleware:
  enabled:
    - type: "timeout"
      config:
        duration: "60s"  # Request-level enforcement timeout
```

#### Server vs Middleware Timeouts

| **Server Timeouts**       | **Middleware Timeouts**        |
| ------------------------- | ------------------------------ |
| Network-level enforcement | Request-level enforcement      |
| OS/TCP stack termination  | Application-level cancellation |
| Lower-level protection    | Higher-level protection        |
| Basic timeout logging     | Detailed timeout reporting     |

> [!TIP]
> Use server timeouts for network protection and middleware timeouts for request-level control. Middleware timeouts should typically be shorter than or equal to server timeouts.

#### Timeout Behavior Example

**Configuration:**
```yaml
middleware:
  enabled:
    - type: "timeout"
      config:
        duration: "200ms"    # Very short timeout for demonstration

routes:
  - path: "/slow"
    method: "GET"
    template: |
      {{ sleep "500ms" }}  # Simulated slow operation
      This response took 500ms to generate
```

**Request and Response:**
```bash
$ curl -i http://localhost:8080/slow

HTTP/1.1 408 Request Timeout
Content-Type: text/plain; charset=utf-8
Date: Mon, 04 Aug 2025 02:36:07 GMT
Content-Length: 121

408 Request Timeout

The request exceeded the configured timeout and was terminated.
Timeout occurred after: 201ms
```

**Server Logs:**
```
level=WARN msg="request timeout - terminating" method=GET path=/slow
  duration=201.089958ms timeout="context cancelled" remote_addr=[::1]:64377
level=INFO msg="request processed" method=GET path=/slow status=408
  duration_ms=201 route=/slow remote_addr=[::1]:64377
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
server:
  timeouts:
    read: "30s"
    write: "30s"
    idle: "120s"
    request: "60s"
    shutdown: "30s"

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

    # Enforce request timeouts
    - type: "timeout"
      config:
        duration: "45s"  # Cancel requests taking longer than 45 seconds

    # Enable request logging (skip health checks)
    - type: "logger"
      config:
        format: "text"
        level: "info"
        skip_paths: ["/health", "/admin/health"]

routes:
  # Public API endpoint
  - path: "/api/data"
    method: "GET"
    template: |
      {
        "message": "Public CORS-enabled API endpoint",
        "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}"
      }
    response_headers:
      Content-Type: "application/json"

  # Protected admin endpoint
  - path: "/admin/dashboard"
    method: "GET"
    template: |
      {
        "message": "Protected admin dashboard",
        "timestamp": "{{ now | date "2006-01-02T15:04:05Z07:00" }}",
        "authenticated": true
      }
    response_headers:
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
- **JSON response** with server information

## Template Syntax

Mockingjay uses Go's [`html/template`](https://pkg.go.dev/html/template) engine with automatic HTML escaping.

### Template Performance

Mockingjay optimizes template execution for high performance:

- **Pre-compilation**: All templates (both inline and file-based) are compiled during server startup and configuration loading
- **No per-request compilation**: Templates are compiled once and reused for all requests, eliminating parsing overhead
- **Hot-reload recompilation**: When configuration changes are detected, all templates are recompiled automatically
- **Validation-time compilation**: The `--validate` command compiles templates to catch syntax errors early
- **Memory efficient**: Compiled templates are stored in memory as `*template.Template` objects ready for immediate execution

This approach ensures minimal latency for request processing while maintaining the flexibility of dynamic templates.

### Template Context

Every template has access to:

```go
{
  "Request": *http.Request,              // Raw HTTP request object
  "Headers": http.Header,                // Request headers with full access to http.Header methods
  "Query":   url.Values,                 // Query parameters with full access to url.Values methods
  "Body":    interface{},                // Parsed JSON body (if applicable)
  "Params":  map[string]string           // URL parameters from regex captures
}
```

### Basic Template Examples

```yaml
template: |
  Request Method: {{ .Request.Method }}
  Request Path: {{ .Request.URL.Path }}
  User Agent: {{ .Headers.Get "User-Agent" }}
  Query Param 'debug': {{ .Query.Get "debug" }}
  All 'tags' values: {{ .Query.Values "tags" | join ", " }}
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

## Template Helper Functions

Mockingjay includes **100+ helper functions** from [Masterminds/sprig](http://masterminds.github.io/sprig/) plus custom functions:

### Custom Functions

| Function       | Description                            | Example                                    |
| -------------- | -------------------------------------- | ------------------------------------------ |
| `trimPrefix`   | Remove prefix from string              | `{{ trimPrefix "/v1" .Request.URL.Path }}` |
| `sleep`        | Introduce delay (for testing)          | `{{ sleep "500ms" }}` or `{{ sleep 2 }}`   |
| `randFloat`    | Generate random floating point number  | `{{ randFloat 12.9 13.7 }}`                |
| `randChoice`   | Randomly select one value from options | `{{ randChoice "red" 1 false }}`           |
| `toJsonPretty` | Multi-line JSON with indentation       | `{{ .Headers \| toJsonPretty }}`           |

### Fake Data Functions

Mockingjay includes **80+ fake data generation functions** powered by [gofakeit](https://github.com/brianvoe/gofakeit) for creating realistic test data:

- **Personal Info**: `fakeName`, `fakeEmail`, `fakePhone`, `fakeAddress`
- **Business Data**: `fakeCompany`, `fakeJobTitle`, `fakeBS`
- **Financial**: `fakeCreditCardNumber`, `fakePrice`, `fakeCurrency`
- **Colors**: `fakeColor`, `fakeHexColor`
- **Internet**: `fakeURL`, `fakeIPv4Address`, `fakeUUID`
- **Text & Words**: `fakeWord`, `fakeWords`, `fakeSentence`, `fakeParagraph`
- **And many more**: Animals, food, entertainment, dates, etc.

The **[complete Fake Data Functions Reference](docs/fake-data-functions.md)** has more information on what functions are available and how to use them.

**Note**: Access headers and query parameters directly using the native Go methods:
- Headers: `{{ .Headers.Get "Content-Type" }}`, `{{ .Headers.Values "Accept" }}`
- Query: `{{ .Query.Get "debug" }}`, `{{ .Query.Values "tags" | join ", " }}`

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

#### JSON Formatting Options
```yaml
template: |
  JSON Formatting Examples:

  Compact: {{ .Headers | toJson }}

  Pretty (multi-line):
  {{ .Headers | toJsonPretty }}
```

**Output comparison:**
- `toJson`: `{"Accept":["application/json"],"User-Agent":["curl/8.7.1"]}`
- `toJsonPretty`: Multi-line with 2-space indentation

## Troubleshooting

### Debug Mode

Enable debug logging to see detailed request information:

```bash
mockingjay --config config.yaml --debug
```

Debug output includes:
- Request details (method, path, headers)
- Route matching process
- Template execution context
- Response generation

### Hot-Reload Support

Mockingjay supports hot-reloading of configuration files:
- **File watching**: Automatically detects changes to the config file
- **Template recompilation**: All templates are recompiled when configuration changes
- **Atomic reloads**: Routes, templates, and middleware are updated atomically
- **Zero downtime**: Server continues serving requests during reload
- **Error handling**: Invalid configurations don't affect running server
- **Thread-safe**: Safe concurrent access during reloads
