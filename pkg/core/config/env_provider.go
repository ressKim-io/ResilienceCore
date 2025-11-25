package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

// EnvProvider implements ConfigProvider for environment variables
type EnvProvider struct {
	prefix string
}

// NewEnvProvider creates a new environment variable configuration provider
// prefix is the prefix for environment variables (e.g., "APP_" will match APP_DATABASE_HOST)
func NewEnvProvider(prefix string) *EnvProvider {
	return &EnvProvider{
		prefix: prefix,
	}
}

// Load loads configuration from environment variables
func (p *EnvProvider) Load(ctx context.Context) (map[string]interface{}, error) {
	values := make(map[string]interface{})

	// Get all environment variables
	environ := os.Environ()

	for _, env := range environ {
		// Split into key=value
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Check if key starts with prefix
		if p.prefix != "" && !strings.HasPrefix(key, p.prefix) {
			continue
		}

		// Remove prefix and convert to lowercase with dots
		configKey := key
		if p.prefix != "" {
			configKey = strings.TrimPrefix(key, p.prefix)
		}

		// Convert underscores to dots and lowercase
		// e.g., DATABASE_HOST -> database.host
		configKey = strings.ToLower(configKey)
		configKey = strings.ReplaceAll(configKey, "_", ".")

		values[configKey] = value
	}

	return values, nil
}

// Watch watches for environment variable changes
// Note: Environment variables don't typically change during runtime,
// so this returns a channel that never sends anything
func (p *EnvProvider) Watch(ctx context.Context) (<-chan types.ConfigChange, error) {
	ch := make(chan types.ConfigChange)

	// Environment variables don't change during runtime in most cases
	// So we just return an empty channel that will be closed when context is done
	go func() {
		<-ctx.Done()
		close(ch)
	}()

	return ch, nil
}

// Name returns the provider name
func (p *EnvProvider) Name() string {
	if p.prefix != "" {
		return fmt.Sprintf("env:%s", p.prefix)
	}
	return "env"
}
