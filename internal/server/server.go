package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/patrickdappollonio/mockingjay/internal/config"
	"github.com/patrickdappollonio/mockingjay/internal/middleware"
	"github.com/patrickdappollonio/mockingjay/internal/router"
	templatepkg "github.com/patrickdappollonio/mockingjay/internal/template"
)

// Server represents the HTTP server with its routes and configuration
type Server struct {
	routes          []*router.Route
	engine          *templatepkg.Engine
	logger          *slog.Logger
	httpServer      *http.Server
	configFile      string        // Path to config file for hot-reload
	mu              sync.RWMutex  // Protects routes and engine during reload
	startTime       time.Time     // Server start time for uptime calculation
	middlewareChain http.Handler  // Middleware chain handler
	shutdownTimeout time.Duration // Configurable shutdown timeout
}

// NewServer creates a new server instance with compiled routes
func NewServer(cfg *config.Config, configFile, addr string, logger *slog.Logger) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Create router compiler and compile routes
	compiler := router.NewCompiler()
	routes, err := compiler.CompileRoutes(cfg.Routes)
	if err != nil {
		return nil, fmt.Errorf("failed to compile routes: %w", err)
	}

	// Get timeout configuration with defaults
	timeouts := cfg.Server.Timeouts.GetWithDefaults()

	server := &Server{
		routes:          routes,
		engine:          compiler.GetEngine(),
		logger:          logger,
		configFile:      configFile,
		startTime:       time.Now(),
		shutdownTimeout: timeouts.Shutdown,
	}

	// Create middleware chain
	middlewareFactory := middleware.NewFactory(logger)
	chain, err := middlewareFactory.CreateChain(cfg.Middleware)
	if err != nil {
		return nil, fmt.Errorf("failed to create middleware chain: %w", err)
	}
	server.middlewareChain = chain.Then(server)

	// Create HTTP server with middleware chain as handler
	server.httpServer = &http.Server{
		Addr:              addr,
		Handler:           server.middlewareChain,
		ReadTimeout:       timeouts.Read,
		WriteTimeout:      timeouts.Write,
		IdleTimeout:       timeouts.Idle,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	return server, nil
}

// ServeHTTP implements the http.Handler interface - main request handler
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Handle built-in health check endpoint
	if r.URL.Path == "/health" && r.Method == http.MethodGet {
		s.handleHealthCheck(w, r)
		s.logRequest(r, 200, time.Since(start), nil)
		return
	}

	// Acquire read lock to ensure thread-safe access to routes and engine
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Find matching route
	routeMatch := s.findMatchingRoute(r)
	if routeMatch == nil {
		s.handleNotFound(w, r)
		s.logRequest(r, 404, time.Since(start), nil)
		return
	}

	// Build template context
	ctx, err := s.engine.BuildTemplateContext(r, routeMatch.Params)
	if err != nil {
		s.handleServerError(w, r, fmt.Errorf("failed to build template context: %w", err))
		s.logRequest(r, 500, time.Since(start), routeMatch.Route)
		return
	}

	// Render custom response headers
	if err := s.renderResponseHeaders(w, routeMatch.Route, ctx); err != nil {
		s.handleTemplateError(w, r, fmt.Errorf("failed to render response headers: %w", err))
		s.logRequest(r, 500, time.Since(start), routeMatch.Route)
		return
	}

	// Execute template
	err = s.engine.ExecuteTemplate(routeMatch.Route.Tmpl, w, ctx)
	if err != nil {
		s.handleTemplateError(w, r, err)
		s.logRequest(r, 500, time.Since(start), routeMatch.Route)
		return
	}

	s.logRequest(r, 200, time.Since(start), routeMatch.Route)
}

// findMatchingRoute iterates through routes to find the first match
func (s *Server) findMatchingRoute(r *http.Request) *router.RouteMatch {
	for _, route := range s.routes {
		if match, ok := route.MatchRequest(r); ok {
			return match
		}
	}
	return nil
}

// handleNotFound handles 404 errors
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404 Not Found: no route matches %s %s", r.Method, r.URL.Path)
}

// handleServerError handles 500 errors
func (s *Server) handleServerError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, "500 Internal Server Error")

	s.logger.Error("server error",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)
}

// handleTemplateError handles template execution errors
func (s *Server) handleTemplateError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, "500 Internal Server Error: Template execution failed")

	s.logger.Error("template execution error",
		"method", r.Method,
		"path", r.URL.Path,
		"error", err,
	)
}

// logRequest logs details about the processed request
func (s *Server) logRequest(r *http.Request, status int, duration time.Duration, route *router.Route) {
	var routePattern string
	if route != nil {
		routePattern = route.Pattern
	} else {
		routePattern = "no match"
	}

	s.logger.Info("request processed",
		"method", r.Method,
		"path", r.URL.Path,
		"status", status,
		"duration_ms", duration.Milliseconds(),
		"route", routePattern,
		"remote_addr", r.RemoteAddr,
	)
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("starting HTTP server",
		"addr", s.httpServer.Addr,
		"routes_count", len(s.routes),
	)

	// Log route details
	for i, route := range s.routes {
		s.logger.Debug("compiled route",
			"index", i,
			"pattern", route.Pattern,
			"verb", route.Verb,
			"is_regex", route.IsRegexp,
			"template_source", route.TemplateSource,
		)
	}

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case <-ctx.Done():
		s.logger.Info("shutting down server", "reason", "context cancelled")
		return s.Shutdown(context.Background())
	case err := <-errCh:
		return fmt.Errorf("server failed to start: %w", err)
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	shutdownCtx, cancel := context.WithTimeout(ctx, s.shutdownTimeout)
	defer cancel()

	s.logger.Info("gracefully shutting down server",
		"timeout", s.shutdownTimeout)
	return s.httpServer.Shutdown(shutdownCtx)
}

// GetAddr returns the server's listening address
func (s *Server) GetAddr() string {
	return s.httpServer.Addr
}

// renderResponseHeaders executes response header templates and sets them on the response
func (s *Server) renderResponseHeaders(w http.ResponseWriter, route *router.Route, ctx *templatepkg.TemplateContext) error {
	// If no custom response headers, nothing to do
	if len(route.ResponseHeaders) == 0 {
		return nil
	}

	// Execute each response header template
	for headerName, headerTemplate := range route.ResponseHeaders {
		var buf bytes.Buffer

		// Execute the header template
		if err := headerTemplate.Execute(&buf, ctx); err != nil {
			return fmt.Errorf("failed to execute template for header %q: %w", headerName, err)
		}

		// Get the rendered header value and trim whitespace
		headerValue := strings.TrimSpace(buf.String())

		// Only set the header if the value is not empty
		if headerValue != "" {
			// Use proper header name capitalization (Go's http package handles this)
			w.Header().Set(headerName, headerValue)
		}
	}

	return nil
}

// ReloadConfig reloads the configuration and recompiles routes
func (s *Server) ReloadConfig() error {
	// Load new configuration
	cfg, err := config.LoadConfig(s.configFile)
	if err != nil {
		return fmt.Errorf("failed to load config during reload: %w", err)
	}

	// Create new router compiler and compile routes
	compiler := router.NewCompiler()
	newRoutes, err := compiler.CompileRoutes(cfg.Routes)
	if err != nil {
		return fmt.Errorf("failed to compile routes during reload: %w", err)
	}

	// Create new middleware chain
	middlewareFactory := middleware.NewFactory(s.logger)
	newChain, err := middlewareFactory.CreateChain(cfg.Middleware)
	if err != nil {
		return fmt.Errorf("failed to create middleware chain during reload: %w", err)
	}
	newMiddlewareChain := newChain.Then(s)

	// Acquire write lock to update routes, engine, and middleware atomically
	s.mu.Lock()
	defer s.mu.Unlock()

	// Update routes, engine, and middleware
	s.routes = newRoutes
	s.engine = compiler.GetEngine()
	s.middlewareChain = newMiddlewareChain

	// Update the HTTP server handler to use the new middleware chain
	s.httpServer.Handler = newMiddlewareChain

	s.logger.Info("configuration reloaded successfully",
		"file", s.configFile,
		"routes_count", len(s.routes),
	)

	// Log new route details in debug mode
	for i, route := range s.routes {
		s.logger.Debug("reloaded route",
			"index", i,
			"pattern", route.Pattern,
			"verb", route.Verb,
			"is_regex", route.IsRegexp,
			"template_source", route.TemplateSource,
		)
	}

	return nil
}

// HealthCheckResponse represents the JSON response for the health check endpoint
type HealthCheckResponse struct {
	Status     string            `json:"status"`
	Version    string            `json:"version"`
	Timestamp  time.Time         `json:"timestamp"`
	Uptime     string            `json:"uptime"`
	Routes     int               `json:"routes"`
	ConfigFile string            `json:"config_file"`
	GoVersion  string            `json:"go_version"`
	Memory     map[string]uint64 `json:"memory"`
}

// handleHealthCheck handles the built-in health check endpoint
func (s *Server) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Calculate uptime
	uptime := time.Since(s.startTime)

	// Get route count (with read lock for thread safety)
	s.mu.RLock()
	routeCount := len(s.routes)
	s.mu.RUnlock()

	// Build response
	response := HealthCheckResponse{
		Status:     "healthy",
		Version:    "dev", // TODO: This could be injected at build time
		Timestamp:  time.Now(),
		Uptime:     uptime.String(),
		Routes:     routeCount,
		ConfigFile: s.configFile,
		GoVersion:  runtime.Version(),
		Memory: map[string]uint64{
			"alloc_bytes":       memStats.Alloc,
			"total_alloc_bytes": memStats.TotalAlloc,
			"sys_bytes":         memStats.Sys,
			"heap_alloc_bytes":  memStats.HeapAlloc,
		},
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Encode and send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		s.logger.Error("failed to encode health check response", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
