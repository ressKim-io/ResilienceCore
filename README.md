# Infrastructure Resilience Engine

> A completely immutable framework for building infrastructure resilience testing applications

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Documentation](https://img.shields.io/badge/docs-latest-brightgreen.svg)](.kiro/specs/infrastructure-resilience-engine/)

[한국어](README.ko.md) | English

## Overview

Infrastructure Resilience Engine is a **completely immutable framework** designed for building infrastructure resilience testing applications. The core principle is **zero Core modification** - all extensions (new environments, plugins, storage backends, monitoring strategies) are added through well-defined interfaces without touching the Core codebase.

**Current Status**: 🚧 In Development - Specification Phase Complete

- ✅ Requirements defined (20 requirements)
- ✅ Design completed (98 correctness properties)
- ✅ Implementation plan ready (6 phases, 50+ tasks)
- 🚧 Phase 1: Core interfaces and models (In Progress)

### Key Features

- 🔒 **Immutable Core**: Add new environments, plugins, and features without modifying Core
- 🌍 **Environment Agnostic**: Unified abstraction for docker-compose, Kubernetes, ECS, Nomad, and more
- 🔌 **Plugin Architecture**: Rich plugin system with lifecycle hooks, rollback support, and dependencies
- 📊 **Comprehensive Observability**: Built-in logging, metrics, and distributed tracing
- 🔐 **Security First**: Authorization, audit logging, and secret management
- 🧪 **Test-Friendly**: Extensive mocking and testing utilities included
- ⚡ **High Performance**: Worker pools, batching, and efficient event streaming

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Core (Immutable)                          │
│                                                              │
│  Interfaces: EnvironmentAdapter, Plugin, ExecutionEngine,   │
│              Monitor, Reporter, EventBus, Config, etc.       │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │ Implements
        ┌───────────────────┴───────────────────┐
        │                                       │
┌───────▼────────┐                    ┌────────▼────────┐
│   Adapters     │                    │    Plugins      │
│                │                    │                 │
│ - Compose      │                    │ - Kill          │
│ - Kubernetes   │                    │ - Restart       │
│ - ECS          │                    │ - Backup        │
│ - Nomad        │                    │ - Scale         │
└────────────────┘                    └─────────────────┘
        │                                       │
        └───────────────────┬───────────────────┘
                            │ Compose
                    ┌───────▼────────┐
                    │  Applications  │
                    │                │
                    │ - GrimOps      │
                    │ - NecroOps     │
                    │ - BackupOps    │
                    └────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Docker (for docker-compose adapter)
- kubectl (for Kubernetes adapter, optional)

### Installation

```bash
go get github.com/yourusername/infrastructure-resilience-engine
```

### Basic Usage

#### 1. Create a Simple Plugin

```go
package main

import (
    "context"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/interfaces"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/types"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Metadata() types.PluginMetadata {
    return types.PluginMetadata{
        Name:        "hello",
        Version:     "1.0.0",
        Description: "A simple hello world plugin",
    }
}

func (p *HelloPlugin) Execute(ctx interfaces.PluginContext, resource types.Resource) error {
    ctx.Logger.Info("Hello from plugin!", 
        types.Field{Key: "resource", Value: resource.Name})
    return nil
}

// Implement other Plugin interface methods...
```

#### 2. Use with an Adapter

```go
package main

import (
    "context"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/adapters/compose"
    "github.com/yourusername/infrastructure-resilience-engine/pkg/core/engine"
)

func main() {
    // Create adapter
    adapter, err := compose.NewComposeAdapter(compose.AdapterConfig{
        ComposeFile: "docker-compose.yml",
    })
    if err != nil {
        panic(err)
    }
    
    // Create execution engine
    engine := engine.NewExecutionEngine()
    
    // Register plugin
    plugin := &HelloPlugin{}
    engine.RegisterPlugin(plugin)
    
    // List resources
    resources, err := adapter.ListResources(context.Background(), types.ResourceFilter{})
    if err != nil {
        panic(err)
    }
    
    // Execute plugin on first resource
    if len(resources) > 0 {
        result, err := engine.Execute(context.Background(), types.ExecutionRequest{
            PluginName: "hello",
            Resource:   resources[0],
        })
        if err != nil {
            panic(err)
        }
        
        fmt.Printf("Execution result: %+v\n", result)
    }
}
```

## Example Applications

### GrimOps (Chaos Engineering)

GrimOps is a chaos engineering application built on the framework:

```bash
# Kill a container
grimops attack redis --plugin kill

# Inject network delay
grimops attack api --plugin network-delay --latency 500ms

# Stress CPU
grimops attack worker --plugin cpu-stress --cores 2 --duration 60s
```

### NecroOps (Self-Healing)

NecroOps is a self-healing application built on the same framework:

```bash
# Watch and auto-heal
necroops heal --watch --config necroops.yaml

# Manual restart
necroops heal redis --plugin restart

# Scale up on failure
necroops heal api --plugin scale --replicas 3
```

## Core Concepts

### Resource Model

The framework uses an environment-agnostic resource model:

```go
type Resource struct {
    ID          string              // Unique identifier
    Name        string              // Human-readable name
    Kind        string              // Resource type (container, pod, task, etc.)
    Labels      map[string]string   // Labels for selection
    Annotations map[string]string   // Additional metadata
    Status      ResourceStatus      // Current status
    Spec        ResourceSpec        // Specification
    Metadata    map[string]interface{} // Environment-specific data
}
```

### Plugin Lifecycle

Plugins follow a rich lifecycle:

```
1. Validate    - Check if execution is possible
2. PreExecute  - Create snapshot for rollback
3. Execute     - Perform the actual operation
4. PostExecute - Post-processing
5. Cleanup     - Always runs, even on failure
6. Rollback    - Optional, runs on failure if supported
```

### Execution Strategies

The framework supports pluggable execution strategies:

- **SimpleStrategy**: Direct execution
- **RetryStrategy**: Retry with exponential backoff
- **CircuitBreakerStrategy**: Prevent cascading failures
- **RateLimitStrategy**: Limit execution rate

### Workflows

Define multi-step workflows with dependencies:

```yaml
workflow:
  name: chaos-and-heal
  steps:
    - name: kill-redis
      plugin: kill
      resource: redis
      
    - name: wait-for-failure
      plugin: wait
      depends_on: [kill-redis]
      
    - name: restart-redis
      plugin: restart
      resource: redis
      depends_on: [wait-for-failure]
      on_error: abort
```

## Development

### Project Structure

```
infrastructure-resilience-engine/
├── cmd/
│   ├── grimops/          # Chaos engineering app
│   └── necroops/         # Self-healing app
├── pkg/
│   ├── core/
│   │   ├── types/        # Data models
│   │   ├── interfaces/   # Core interfaces
│   │   ├── engine/       # Execution engine
│   │   ├── monitor/      # Monitoring system
│   │   ├── reporter/     # Reporting system
│   │   ├── eventbus/     # Event bus
│   │   ├── config/       # Configuration
│   │   ├── registry/     # Plugin registry
│   │   └── testing/      # Test utilities
│   ├── adapters/
│   │   ├── compose/      # docker-compose adapter
│   │   └── k8s/          # Kubernetes adapter
│   └── plugins/
│       ├── kill/         # Kill plugin
│       ├── restart/      # Restart plugin
│       ├── delay/        # Network delay plugin
│       └── healthmonitor/ # Health monitor plugin
├── docs/
│   ├── plugin-development.md
│   ├── adapter-development.md
│   └── api-reference.md
├── examples/
│   ├── simple-plugin/
│   ├── custom-adapter/
│   └── workflow/
└── .kiro/specs/infrastructure-resilience-engine/
    ├── requirements.md
    ├── design.md
    └── tasks.md
```

### Building from Source

```bash
# Clone the repository
git clone https://github.com/yourusername/infrastructure-resilience-engine.git
cd infrastructure-resilience-engine

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run with race detector
make test-race

# Lint
make lint
```

### Running Tests

```bash
# Unit tests
go test ./...

# Integration tests
go test -tags=integration ./...

# Property-based tests
go test -tags=property ./...

# Coverage
go test -cover ./...
```

## Documentation

- [Requirements](.kiro/specs/infrastructure-resilience-engine/requirements.md)
- [Design](.kiro/specs/infrastructure-resilience-engine/design.md)
- [Plugin Development Guide](docs/plugin-development.md)
- [Adapter Development Guide](docs/adapter-development.md)
- [API Reference](docs/api-reference.md)

## Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details.

### Development Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Commit your changes (`git commit -m 'feat: add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Code Style

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` and `goimports`
- Write tests for new features
- Document public APIs

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Inspired by Chaos Mesh, Pumba, and LitmusChaos
- Built for the Kiroween Hackathon
- Special thanks to the DevOps community

## Contact

- GitHub Issues: [https://github.com/yourusername/infrastructure-resilience-engine/issues](https://github.com/yourusername/infrastructure-resilience-engine/issues)
- Email: your.email@example.com

---

**Built with ❤️ for infrastructure resilience testing**
