package middleware

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/justinas/alice"
)

// Config represents middleware configuration from YAML
type Config struct {
	Enabled []MiddlewareConfig `yaml:"enabled"`
}

// MiddlewareConfig represents a single middleware configuration
type MiddlewareConfig struct {
	Type   string                 `yaml:"type"`   // "cors", "logger", etc.
	Config map[string]interface{} `yaml:"config"` // Type-specific configuration
}

// Factory creates middleware instances from configuration
type Factory struct {
	logger *slog.Logger
}

// NewFactory creates a new middleware factory
func NewFactory(logger *slog.Logger) *Factory {
	return &Factory{logger: logger}
}

// CreateMiddleware creates a middleware instance from configuration
func (f *Factory) CreateMiddleware(config MiddlewareConfig) (Middleware, error) {
	switch config.Type {
	case "cors":
		return f.createCORSMiddleware(config.Config)
	case "logger":
		return f.createLoggerMiddleware(config.Config)
	case "basicauth":
		return f.createBasicAuthMiddleware(config.Config)
	case "timeout":
		return f.createTimeoutMiddleware(config.Config)
	default:
		return nil, fmt.Errorf("unknown middleware type %q", config.Type)
	}
}

// CreateChain creates a middleware chain from configuration
func (f *Factory) CreateChain(config Config) (alice.Chain, error) {
	var middlewares []Middleware

	for _, middlewareConfig := range config.Enabled {
		middleware, err := f.CreateMiddleware(middlewareConfig)
		if err != nil {
			return alice.Chain{}, fmt.Errorf("failed to create middleware %s: %w", middlewareConfig.Type, err)
		}
		middlewares = append(middlewares, middleware)
	}

	return NewChain(middlewares...), nil
}

// createCORSMiddleware creates CORS middleware from config map
func (f *Factory) createCORSMiddleware(configMap map[string]interface{}) (Middleware, error) {
	config := CORSConfig{}

	// Parse configuration with type assertions
	if origins, ok := configMap["allow_origins"].([]interface{}); ok {
		config.AllowOrigins = make([]string, len(origins))
		for i, origin := range origins {
			if str, ok := origin.(string); ok {
				config.AllowOrigins[i] = str
			}
		}
	}

	if methods, ok := configMap["allow_methods"].([]interface{}); ok {
		config.AllowMethods = make([]string, len(methods))
		for i, method := range methods {
			if str, ok := method.(string); ok {
				config.AllowMethods[i] = str
			}
		}
	}

	if headers, ok := configMap["allow_headers"].([]interface{}); ok {
		config.AllowHeaders = make([]string, len(headers))
		for i, header := range headers {
			if str, ok := header.(string); ok {
				config.AllowHeaders[i] = str
			}
		}
	}

	if credentials, ok := configMap["allow_credentials"].(bool); ok {
		config.AllowCredentials = credentials
	}

	if maxAge, ok := configMap["max_age"].(int); ok {
		config.MaxAge = maxAge
	}

	return NewCORSMiddleware(config), nil
}

// createLoggerMiddleware creates logger middleware from config map
func (f *Factory) createLoggerMiddleware(configMap map[string]interface{}) (Middleware, error) {
	config := LoggerConfig{}

	if format, ok := configMap["format"].(string); ok {
		config.Format = format
	}

	if level, ok := configMap["level"].(string); ok {
		config.Level = level
	}

	if skipPaths, ok := configMap["skip_paths"].([]interface{}); ok {
		config.SkipPaths = make([]string, len(skipPaths))
		for i, path := range skipPaths {
			if str, ok := path.(string); ok {
				config.SkipPaths[i] = str
			}
		}
	}

	return NewLoggerMiddleware(f.logger, config), nil
}

// createBasicAuthMiddleware creates basic auth middleware from config map
func (f *Factory) createBasicAuthMiddleware(configMap map[string]interface{}) (Middleware, error) {
	config := BasicAuthConfig{}

	if username, ok := configMap["username"].(string); ok {
		config.Username = username
	}

	if password, ok := configMap["password"].(string); ok {
		config.Password = password
	}

	if realm, ok := configMap["realm"].(string); ok {
		config.Realm = realm
	}

	// Parse paths configuration
	if pathsMap, ok := configMap["paths"].(map[string]interface{}); ok {
		if includeList, ok := pathsMap["include"].([]interface{}); ok {
			config.Paths.Include = make([]string, len(includeList))
			for i, path := range includeList {
				if str, ok := path.(string); ok {
					config.Paths.Include[i] = str
				}
			}
		}

		if excludeList, ok := pathsMap["exclude"].([]interface{}); ok {
			config.Paths.Exclude = make([]string, len(excludeList))
			for i, path := range excludeList {
				if str, ok := path.(string); ok {
					config.Paths.Exclude[i] = str
				}
			}
		}
	}

	// Validate required fields
	if config.Username == "" {
		return nil, fmt.Errorf("basic auth username is required")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("basic auth password is required")
	}

	return NewBasicAuthMiddleware(config)
}

// createTimeoutMiddleware creates timeout middleware from config map
func (f *Factory) createTimeoutMiddleware(configMap map[string]interface{}) (Middleware, error) {
	config := TimeoutConfig{}

	// Parse duration from string or integer (seconds)
	if duration, ok := configMap["duration"].(string); ok {
		if parsed, err := time.ParseDuration(duration); err == nil {
			config.Duration = parsed
		} else {
			return nil, fmt.Errorf("invalid timeout duration format: %v", err)
		}
	} else if seconds, ok := configMap["duration"].(int); ok {
		config.Duration = time.Duration(seconds) * time.Second
	} else if seconds, ok := configMap["duration"].(float64); ok {
		config.Duration = time.Duration(seconds) * time.Second
	}

	return NewTimeoutMiddleware(config, f.logger), nil
}
