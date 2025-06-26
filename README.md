# MCPBlade

MCPBlade is a powerful aggregation and management service for Model Control Protocol (MCP) servers. It provides intelligent tool discovery, semantic search, and smart routing capabilities to seamlessly work with multiple MCP servers.

## Features

- **Tool Aggregation**: Combines tools from multiple MCP servers into a unified interface
- **Semantic Search**: Vector-based search to find tools using natural language queries
- **Smart Routing**: Automatically routes tool calls to the correct backend server
- **Health Monitoring**: Continuous health checks and heartbeat monitoring for all connected servers
- **Persistent & Temporary Servers**: Support for both persistent configuration-based and temporary runtime servers
- **Multiple Transport Types**: Support for stdio, SSE, and streamable HTTP transports for backend connections
- **NATS Integration**: Distributed service communication via NATS messaging with microservices support
- **HTTP API**: RESTful API and MCP-compatible streaming endpoints
- **Middleware Architecture**: Logging, proxy, and other middleware support

## Architecture

MCPBlade consists of several key components:

- **Core Service** ([`service.go`](service.go)): Main business logic and MCP server management
- **Vector Search** ([`vector/`](vector/)): Tool indexing and semantic search capabilities  
- **Transport Layer**: 
  - [`transport/nats/`](transport/nats/): NATS-based communication
  - [`transport/http/`](transport/http/): HTTP API endpoints
- **Persistence** ([`persistence/chromem/`](persistence/chromem/)): Vector database implementation
- **MCP Integration** ([`mcp/`](mcp/)): MCP protocol endpoint handlers
- **Proxy Layer** ([`proxy.go`](proxy.go)): Service proxy middleware for distributed deployments

## Installation

```bash
go install github.com/flarexio/mcpblade/cmd/mcpblade@latest
go install github.com/flarexio/mcpblade/cmd/mcpblade_mcp_server@latest
```

## Configuration

Create a configuration file `config.yaml` (see [`config.example.yaml`](config.example.yaml) for reference):

```yaml
mcpServers:
  time:
    transport: stdio
    command: uvx
    args: [ "mcp-server-time", "--local-timezone=Asia/Taipei" ]
cacheRefreshTTL: 5m
vector:
  enabled: true
  persistent: true
  collection: tools
```

### Supported Transport Types

- **stdio**: Standard input/output communication with subprocess
- **sse**: Server-Sent Events over HTTP
- **streamable-http**: HTTP streaming protocol

## Usage

### Running the Core Service

```bash
# Run with default configuration path (~/.flarex/mcpblade)
mcpblade

# Run with custom configuration path
mcpblade --path /path/to/config

# Enable HTTP API
mcpblade --http --http-addr :8080

# Connect to custom NATS server
mcpblade --nats nats://localhost:4222
```

### Running as MCP Server

```bash
# Connect to shared MCPBlade instance
mcpblade_mcp_server --edge-id your-edge-id

# Run dedicated server with specific MCP backend
mcpblade_mcp_server --edge-id your-edge-id --server-id my-server --cmd "uvx mcp-server-time"

# Connect to custom NATS server
mcpblade_mcp_server --edge-id your-edge-id --nats nats://localhost:4222
```

## API Reference

### Core Service Interface

The [`Service`](service.go) interface provides these methods:

- **RegisterMCPServer**: Add a new MCP server to the registry
- **UnregisterMCPServer**: Remove an MCP server from the registry  
- **ListTools**: Get all available tools from registered servers
- **SearchTools**: Search for tools using semantic queries
- **Forward**: Route MCP requests to appropriate backend servers
- **Close**: Gracefully shutdown the service

### HTTP API Endpoints

When HTTP transport is enabled:

```bash
# RESTful API
POST   /api/mcp/register           # Register MCP server
DELETE /api/mcp/unregister/:id     # Unregister MCP server
GET    /api/mcp/tools              # List all tools
GET    /api/mcp/tools/search       # Search tools
POST   /api/mcp/forward            # Forward tool calls

# MCP Protocol
POST   /mcp/                       # MCP JSON-RPC endpoint
```

### Example Tool Search

```go
// Semantic search for tools
tools, err := service.SearchTools(ctx, "what's the current time?", 5)
if err != nil {
    return err
}

for _, tool := range tools {
    fmt.Printf("Tool: %s - %s\n", tool.Name, tool.Description)
}
```

### Example Tool Call

```go
req := mcp.CallToolRequest{
    Request: mcp.Request{Method: "tools/call"},
    Params: mcp.CallToolParams{
        Name: "get_current_time",
        Arguments: map[string]any{
            "timezone": "Asia/Taipei",
        },
    },
}

result, err := service.Forward(ctx, req)
if err != nil {
    return err
}

// Handle result
if !result.IsError {
    for _, content := range result.Content {
        fmt.Printf("Result: %v\n", content)
    }
}
```

## Transport Architecture

```
[MCP Clients] 
    ↓ (stdio/nats/http)
[MCPBlade Proxy/Service]
    ↓ (stdio/sse/streamable-http)  
[Backend MCP Servers]
```

### Flow Description

1. **Inbound**: Clients connect to MCPBlade via:
   - **stdio**: Native MCP protocol (via [`mcpblade_mcp_server`](cmd/mcpblade_mcp_server/main.go))
   - **nats**: Distributed messaging (via [`transport/nats/`](transport/nats/))
   - **http**: RESTful API and MCP streaming (via [`transport/http/`](transport/http/))

2. **Processing**: MCPBlade aggregates, searches, and routes requests using:
   - Tool caching and deduplication
   - Vector-based semantic search  
   - Health monitoring and heartbeat tracking

3. **Outbound**: MCPBlade connects to backend servers as an MCP client via:
   - **stdio**: Subprocess communication
   - **sse**: Server-Sent Events
   - **streamable-http**: HTTP streaming

## Vector Search

Tools are automatically indexed in a vector database for intelligent search:

```go
// Search finds tools using semantic similarity
tools, err := service.SearchTools(ctx, "time and date functions", 10)

// Tools are indexed with metadata including:
// - Server ID
// - Tool name and description
// - Input schema
// - Full tool JSON for reconstruction
```

The vector implementation uses [ChromeDB](https://github.com/philippgille/chromem-go) with support for:
- In-memory collections for development
- Persistent storage for production
- Automatic embedding generation
- Similarity search with configurable result limits

## Health Monitoring

The service continuously monitors the health of all registered MCP servers:

- **Periodic Health Checks**: Configurable TTL-based monitoring
- **Heartbeat Tracking**: Atomic timestamp tracking for each server
- **Automatic Recovery**: Failed servers can be automatically restarted
- **Cache Refresh**: Tool cache is refreshed based on health status
- **Graceful Degradation**: Failed servers are excluded from routing

## Middleware System

MCPBlade uses a middleware architecture for cross-cutting concerns:

### Logging Middleware

```go
svc = mcpblade.LoggingMiddleware(logger)(svc)
```

Provides structured logging with [Zap](https://github.com/uber-go/zap) for all service operations.

### Proxy Middleware  

```go
svc = mcpblade.ProxyMiddleware(endpoints)(svc)
```

Enables distributed deployments by proxying calls through NATS endpoints.

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run specific test suite
go test ./service_test.go
```

### Project Structure

```
├── cmd/
│   ├── mcpblade/                    # Main service executable
│   └── mcpblade_mcp_server/         # MCP server implementation
├── transport/
│   ├── nats/                        # NATS transport layer
│   │   ├── transport.go             # NATS handlers
│   │   ├── factory.go               # NATS client endpoints
│   │   └── topic.go                 # NATS topic routing
│   └── http/                        # HTTP transport layer
│       ├── transport.go             # HTTP handlers
│       ├── mcp.go                   # MCP streaming endpoints
│       └── router.go                # Route configuration
├── persistence/chromem/             # ChromeDB vector database
├── vector/                          # Vector database interfaces
├── mcp/                            # MCP protocol handlers
│   ├── endpoint.go                  # MCP endpoint implementations
│   └── endpoint_test.go             # MCP protocol tests
├── service.go                       # Core service implementation
├── service_test.go                  # Service integration tests
├── model.go                         # Data models and types
├── model_test.go                    # Model serialization tests
├── endpoint.go                      # Go-kit endpoints
├── proxy.go                         # Proxy middleware
├── logging.go                       # Logging middleware
└── config.example.yaml              # Example configuration
```

## Dependencies

- [mcp-go](https://github.com/mark3labs/mcp-go): MCP protocol implementation
- [chromem-go](https://github.com/philippgille/chromem-go): Vector database
- [nats.go](https://github.com/nats-io/nats.go): NATS messaging with microservices
- [go-kit](https://github.com/go-kit/kit): Microservice toolkit for endpoints
- [gin](https://github.com/gin-gonic/gin): HTTP web framework
- [zap](https://github.com/uber-go/zap): Structured logging
- [cli/v3](https://github.com/urfave/cli): Command-line interface

## License

MIT License - see [LICENSE.md](LICENSE.md) for details.

This project is part of the FlareX ecosystem.

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code follows Go conventions
3. New features include tests
4. Documentation is updated
5. Commit messages are descriptive

For major changes, please open an issue first to discuss the proposed changes.
