package mcpblade

import (
	"context"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

// Service defines the core logic of MCPBlade.
type Service interface {

	// RegisterMCPServer adds a new MCP server to the registry.
	RegisterMCPServer(ctx context.Context, id string, config MCPServerConfig, persistnt ...bool) error

	// // UnregisterMCPServer removes an MCP server from the registry.
	// UnregisterMCPServer(ctx context.Context, serverID string) error

	// // Heartbeat updates the liveness timestamp of an MCP server.
	// Heartbeat(ctx context.Context, serverID string) error

	// ListTools returns the aggregated tool list (deduplicated),
	ListTools(ctx context.Context) ([]mcp.Tool, error)

	// // SearchTools searches for tools matching the given query.
	// SearchTools(ctx context.Context, query string) ([]ToolInfo, error)

	// // Forward routes an MCP protocol request to an appropriate backend MCP server.
	// Forward(ctx context.Context, tool string, req *api.Request) (*api.Response, error)

	// // Stream handles long-running or streaming requests.
	// // It forwards the stream from the backend MCP server to the client via StreamWriter.
	// Stream(ctx context.Context, tool string, req *api.Request, writer StreamWriter) error

	// // GetMCPServerStatus returns the current registration & heartbeat status.
	// GetMCPServerStatus(ctx context.Context) ([]ServerStatus, error)
}

func NewService(ctx context.Context, cfg Config) Service {
	log := zap.L().With(
		zap.String("service", "mcpblade"),
	)

	svc := &service{
		ctx:                 ctx,
		log:                 log,
		persistentInstances: make(map[string]client.MCPClient),
		temporaryInstances:  make(map[string]client.MCPClient),
		toolRoutes:          make(map[string]string),
		toolsCache:          make([]mcp.Tool, 0),
	}

	// Register persistent MCP servers
	for id, config := range cfg.MCPServers {
		log := log.With(
			zap.String("server_id", id),
		)

		err := svc.RegisterMCPServer(ctx, id, config, true)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		log.Info("registered persistent MCP server")
	}

	// Refresh tools cache immediately on startup
	svc.refreshToolsCache(ctx)

	// Start the tools cache refresher worker
	go svc.runToolsCacheWorker(ctx, cfg.CacheRefreshTTL)

	return svc
}

type service struct {
	ctx context.Context
	log *zap.Logger

	persistentInstances map[string]client.MCPClient // Persistent MCP servers that are always running
	temporaryInstances  map[string]client.MCPClient // Temporary MCP servers that are created on-demand
	toolRoutes          map[string]string           // Maps tool names to server IDs
	toolsCache          []mcp.Tool                  // Cached tools for quick access
	sync.RWMutex
}

func (svc *service) RegisterMCPServer(ctx context.Context, id string, config MCPServerConfig, persistent ...bool) error {
	isPersistent := false
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	if id == "" {
		return ErrInvalidServerID
	}

	svc.Lock()
	defer svc.Unlock()

	var instances map[string]client.MCPClient
	if isPersistent {
		instances = svc.persistentInstances
	} else {
		instances = svc.temporaryInstances
	}

	_, ok := instances[id]
	if ok {
		return ErrServerAlreadyExists
	}

	var (
		c   *client.Client
		err error
	)

	switch config.Transport {
	case TransportTypeStdio:
		c, err = client.NewStdioMCPClient(
			config.Command,
			config.Environment,
			config.Arguments...,
		)

	case TransportTypeSSE:
		c, err = client.NewSSEMCPClient(config.URL)

	case TransportTypeStreamableHTTP:
		c, err = client.NewStreamableHttpClient(config.URL)

	default:
		return ErrUnsupportedTransportType
	}

	if err != nil {
		return err
	}

	if err := c.Start(ctx); err != nil {
		return err
	}

	req := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcpblade",
				Version: "1.0.0",
			},
		},
	}

	if _, err := c.Initialize(ctx, req); err != nil {
		return err
	}

	instances[id] = c

	return nil
}

func (svc *service) runToolsCacheWorker(ctx context.Context, ttl time.Duration) {
	log := svc.log.With(
		zap.String("action", "start_tools_cache_refresher"),
		zap.Duration("ttl", ttl),
	)

	ticker := time.NewTicker(ttl)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("done")
			return

		case <-ticker.C:
			log.Info("refreshing tools cache")
			svc.refreshToolsCache(ctx)
		}
	}
}

func (svc *service) refreshToolsCache(ctx context.Context) {
	log := svc.log.With(
		zap.String("action", "refresh_tools_cache"),
	)

	var (
		routes = make(map[string]string)
		tools  = make([]mcp.Tool, 0)
	)

	for id, client := range svc.persistentInstances {
		log := log.With(
			zap.String("server_id", id),
		)

		var cursor mcp.Cursor
		for {
			req := mcp.ListToolsRequest{
				PaginatedRequest: mcp.PaginatedRequest{
					Params: mcp.PaginatedParams{
						Cursor: cursor,
					},
				},
			}

			results, err := client.ListTools(ctx, req)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			for _, tool := range results.Tools {
				log := log.With(
					zap.String("tool", tool.Name),
				)

				if tool.Description != "" {
					tool.Description = tool.Description + " (provided by " + id + ")"
				} else {
					tool.Description = "Provided by " + id
				}

				if _, ok := routes[tool.Name]; ok {
					log.Warn("duplicate tool name found")

					tool.Name = id + ":" + tool.Name
					routes[tool.Name] = id
				} else {
					routes[tool.Name] = id
				}

				tools = append(tools, tool)
			}

			cursor = results.NextCursor
			if cursor == "" {
				break
			}
		}
	}

	if len(tools) == 0 {
		log.Error(ErrNoToolsFound.Error())
	}

	svc.Lock()
	svc.toolRoutes = routes
	svc.toolsCache = tools
	svc.Unlock()

	log.Info("tools cache refreshed",
		zap.Int("count", len(tools)))
}

func (svc *service) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	svc.RLock()
	defer svc.RUnlock()

	// Check if a specific server ID is provided in the context
	serverID, ok := ctx.Value(ServerID).(string)
	if !ok {
		// Aggregated mode: return cached tools from all persistent servers
		if len(svc.toolsCache) == 0 {
			return nil, ErrNoToolsFound
		}

		tools := make([]mcp.Tool, len(svc.toolsCache))
		copy(tools, svc.toolsCache)

		return tools, nil
	}

	// Specific server mode: get tools from the specified temporary server
	client, ok := svc.temporaryInstances[serverID]
	if !ok {
		return nil, ErrServerNotFound
	}

	var (
		cursor mcp.Cursor
		tools  []mcp.Tool
	)

	for {
		req := mcp.ListToolsRequest{
			PaginatedRequest: mcp.PaginatedRequest{
				Params: mcp.PaginatedParams{
					Cursor: cursor,
				},
			},
		}

		results, err := client.ListTools(ctx, req)
		if err != nil {
			return nil, err
		}

		tools = append(tools, results.Tools...)

		cursor = results.NextCursor
		if cursor == "" {
			break
		}
	}

	if len(tools) == 0 {
		return nil, ErrNoToolsFound
	}

	return tools, nil
}
