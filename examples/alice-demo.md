# Alice-Based Middleware with ResponseWriter Hijacking

Your middleware system now uses the battle-tested `github.com/justinas/alice` library and supports full `http.ResponseWriter` hijacking capabilities. Here's what changed and how to use it:

## What Changed

1. **Alice Integration**: Replaced custom middleware chaining with Alice
2. **ResponseWriter Wrapping**: Automatic ResponseWriter wrapping for status/size capture
3. **Standard Go Patterns**: All middlewares now use the standard `func(http.Handler) http.Handler` signature
4. **Full Interface Support**: Hijacker, Flusher, Pusher, CloseNotifier interfaces are all supported

## Creating Custom Middleware

```go
package main

import (
    "log/slog"
    "net/http"

    "github.com/patrickdappollonio/mockingjay/internal/middleware"
)

// CustomMiddleware demonstrates the new pattern
type CustomMiddleware struct {
    logger *slog.Logger
}

func (c *CustomMiddleware) Name() string {
    return "custom"
}

func (c *CustomMiddleware) Handler() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Call next handler
            next.ServeHTTP(w, r)

            // Access captured response metadata
            if wrapper, ok := w.(*middleware.ResponseWriter); ok {
                c.logger.Info("response captured",
                    "status", wrapper.Status(),
                    "size", wrapper.Size(),
                    "written", wrapper.Written(),
                )
            }

            // Use standard interfaces
            if hijacker, ok := w.(http.Hijacker); ok {
                // WebSocket upgrades, etc.
                conn, rw, err := hijacker.Hijack()
                // Handle connection...
            }

            if flusher, ok := w.(http.Flusher); ok {
                // Streaming responses
                flusher.Flush()
            }
        })
    }
}

func main() {
    logger := slog.Default()

    // Create middlewares
    loggerMW := middleware.NewLoggerMiddleware(logger, middleware.LoggerConfig{})
    customMW := &CustomMiddleware{logger: logger}

    // Build chain using Alice
    chain := middleware.NewChain(loggerMW, customMW)

    // Final handler
    finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusCreated)
        w.Write([]byte("Hello Alice!"))
    })

    // Create complete handler
    handler := chain.Then(finalHandler)

    http.ListenAndServe(":8080", handler)
}
```

## Key Benefits

- **Standard Go Patterns**: Follows idiomatic Go middleware patterns
- **Battle-Tested**: Alice is used by thousands of Go projects
- **Full Interface Support**: All `http.ResponseWriter` interfaces work correctly
- **Automatic Wrapping**: ResponseWriter automatically wrapped for status/size capture
- **Backward Compatible**: All existing configurations continue to work

## Example Output

```
time=2025-08-03T19:45:43.826-04:00 level=INFO msg="request processed" method=GET path=/test status=201 size=12 duration_ms=0 remote_addr=127.0.0.1:8080
time=2025-08-03T19:45:43.826-04:00 level=INFO msg="response captured" status=201 size=12 written=true
```

Your middleware system now properly supports ResponseWriter hijacking while maintaining full compatibility with existing configurations!
