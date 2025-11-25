# Config Package

The `config` package provides a flexible configuration management system with support for multiple configuration sources, precedence-based merging, type conversion, and hot-reloading.

## Features

- **Multiple Configuration Sources**: Support for YAML files, environment variables, and custom providers
- **Priority-Based Merging**: Higher priority sources override lower priority ones
- **Type Conversion**: Automatic conversion between types (string, int, bool, duration)
- **Hot-Reloading**: Watch for configuration changes and notify subscribers
- **Thread-Safe**: All operations are safe for concurrent use

## Usage

### Basic Usage

```go
import "github.com/yourusername/infrastructure-resilience-engine/pkg/core/config"

// Create a new config instance
cfg := config.NewDefaultConfig()
defer cfg.Close()

// Set and get values
cfg.Set("app.name", "my-app")
name, _ := cfg.GetString("app.name")

// Type conversion
cfg.Set("app.port", 8080)
port, _ := cfg.GetInt("app.port")

cfg.Set("app.debug", true)
debug, _ := cfg.GetBool("app.debug")

cfg.Set("app.timeout", "30s")
timeout, _ := cfg.GetDuration("app.timeout")
```

### Using Configuration Providers

#### YAML Provider

```go
// Load configuration from YAML file
yamlProvider := config.NewYAMLProvider("config.yaml")
cfg.AddProvider(yamlProvider, 50) // Priority 50

// config.yaml:
// database:
//   host: localhost
//   port: 5432
// app:
//   name: my-app
//   debug: true

// Access flattened keys
host, _ := cfg.GetString("database.host")
port, _ := cfg.GetInt("database.port")
```

#### Environment Variable Provider

```go
// Load configuration from environment variables
// Variables with prefix "APP_" will be loaded
envProvider := config.NewEnvProvider("APP_")
cfg.AddProvider(envProvider, 100) // Priority 100 (highest)

// Environment variables:
// APP_DATABASE_HOST=localhost
// APP_DATABASE_PORT=5432
// APP_DEBUG=true

// Access with converted keys (underscores to dots, lowercase)
host, _ := cfg.GetString("database.host")
port, _ := cfg.GetInt("database.port")
debug, _ := cfg.GetBool("debug")
```

### Priority-Based Merging

Configuration sources are merged based on priority. Higher priority values override lower priority values.

```go
cfg := config.NewDefaultConfig()
defer cfg.Close()

// Add YAML provider (priority 50)
yamlProvider := config.NewYAMLProvider("config.yaml")
cfg.AddProvider(yamlProvider, 50)

// Add environment provider (priority 100)
envProvider := config.NewEnvProvider("APP_")
cfg.AddProvider(envProvider, 100)

// If both providers have the same key, environment variable wins
// because it has higher priority
```

### Watching for Changes

```go
cfg := config.NewDefaultConfig()
defer cfg.Close()

// Watch for changes to a specific key
ch, _ := cfg.Watch("app.debug")

go func() {
    for value := range ch {
        fmt.Printf("Config changed: app.debug = %v\n", value)
    }
}()

// Change the value
cfg.Set("app.debug", false) // Watcher will be notified
```

### Custom Configuration Provider

You can implement custom configuration providers by implementing the `ConfigProvider` interface:

```go
type ConfigProvider interface {
    Load(ctx context.Context) (map[string]interface{}, error)
    Watch(ctx context.Context) (<-chan ConfigChange, error)
    Name() string
}
```

Example:

```go
type MyCustomProvider struct {
    // Your fields
}

func (p *MyCustomProvider) Load(ctx context.Context) (map[string]interface{}, error) {
    // Load configuration from your source
    return map[string]interface{}{
        "key1": "value1",
        "key2": 42,
    }, nil
}

func (p *MyCustomProvider) Watch(ctx context.Context) (<-chan types.ConfigChange, error) {
    ch := make(chan types.ConfigChange)
    // Implement watching logic or return empty channel
    return ch, nil
}

func (p *MyCustomProvider) Name() string {
    return "my-custom-provider"
}

// Use it
cfg := config.NewDefaultConfig()
cfg.AddProvider(&MyCustomProvider{}, 75)
```

## Type Conversion

The config system automatically converts between types:

| From Type | To Type | Conversion |
|-----------|---------|------------|
| string | int | `strconv.Atoi()` |
| string | bool | `strconv.ParseBool()` |
| string | duration | `time.ParseDuration()` |
| int | string | `fmt.Sprintf("%d", v)` |
| bool | string | `fmt.Sprintf("%v", v)` |
| int | bool | `v != 0` |

## Thread Safety

All operations on `DefaultConfig` are thread-safe and can be called from multiple goroutines concurrently.

## Best Practices

1. **Use Priority Wisely**: Assign priorities that make sense for your use case
   - Environment variables: 100 (highest - for overrides)
   - Command-line flags: 90
   - Config files: 50
   - Defaults: 10 (lowest)

2. **Close Config**: Always call `Close()` when done to clean up watchers and goroutines

3. **Handle Errors**: Always check errors from Get methods, as keys may not exist

4. **Use Type-Specific Methods**: Use `GetInt()`, `GetBool()`, etc. instead of `Get()` for better type safety

5. **Flatten YAML Keys**: YAML nested structures are automatically flattened with dot notation
   ```yaml
   database:
     host: localhost
   ```
   Access as: `cfg.GetString("database.host")`

## Testing

The package includes comprehensive unit tests and property-based tests:

```bash
# Run all tests
go test ./pkg/core/config/...

# Run only unit tests
go test ./pkg/core/config/... -run "^Test[^P]"

# Run only property-based tests
go test ./pkg/core/config/... -run "TestProperty"
```

## Examples

See `config_test.go` for more examples of usage patterns.
