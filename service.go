package mcpblade

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"

	"github.com/flarexio/mcpblade/vector"
)

// Service defines the core logic of MCPBlade.
type Service interface {

	// Close gracefully shuts down the service and all registered MCP servers.
	Close() error

	// RegisterMCPServer adds a new MCP server to the registry.
	RegisterMCPServer(ctx context.Context, id string, config MCPServerConfig, persistent ...bool) error

	// UnregisterMCPServer removes an MCP server from the registry.
	UnregisterMCPServer(ctx context.Context, serverID string, persistent ...bool) error

	// ListTools returns the aggregated tool list (deduplicated),
	ListTools(ctx context.Context) ([]mcp.Tool, error)

	// SearchTools searches for tools matching the given query.
	SearchTools(ctx context.Context, query string, k ...int) ([]mcp.Tool, error)

	// Forward routes an MCP protocol request to an appropriate backend MCP server.
	Forward(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error)
}

type ServiceMiddleware func(Service) Service

func NewService(ctx context.Context, cfg Config, vector vector.VectorDB) (Service, error) {
	log := zap.L().With(
		zap.String("service", "mcpblade"),
	)

	ctx, cancel := context.WithCancel(ctx)

	svc := &service{
		persistentInstances: make(map[string]*MCPServerInstance),
		temporaryInstances:  make(map[string]*MCPServerInstance),
		toolRoutes:          make(map[string]string),
		toolsCache:          make([]mcp.Tool, 0),

		cfg:    cfg,
		log:    log,
		ctx:    ctx,
		cancel: cancel,
	}

	if vector != nil {
		collection, err := vector.Collection(cfg.Vector.Collection)
		if err != nil {
			return nil, err
		}

		svc.collection = collection
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

	svc.cacheTools(ctx)

	go svc.healthMonitor(ctx, cfg.CacheRefreshTTL)

	return svc, nil
}

type service struct {
	// Persistent instances and their protection
	persistentInstances map[string]*MCPServerInstance

	// Temporary instances and their protection
	temporaryInstances map[string]*MCPServerInstance
	temporaryMutex     sync.RWMutex

	// Tools cache and routing
	toolRoutes map[string]string
	toolsCache []mcp.Tool

	// Vector collection (thread-safe by itself)
	collection vector.Collection

	cfg    Config
	log    *zap.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

func (svc *service) Close() error {
	log := svc.log.With(
		zap.String("action", "close"),
	)

	if svc.cancel != nil {
		svc.cancel()
		svc.cancel = nil
	}

	// Close all persistent MCP clients
	for id, instance := range svc.persistentInstances {
		log := log.With(
			zap.String("server_id", id),
			zap.String("type", "persistent"),
		)

		if err := instance.Client.Close(); err != nil {
			log.Error(err.Error())
			continue
		}

		log.Info("closed persistent MCP client")
	}

	// Close all temporary MCP clients
	svc.temporaryMutex.Lock()
	for id, instance := range svc.temporaryInstances {
		log := log.With(
			zap.String("server_id", id),
			zap.String("type", "temporary"),
		)

		if err := instance.Client.Close(); err != nil {
			log.Error(err.Error())
			continue
		}

		log.Info("closed temporary MCP client")
	}

	svc.temporaryInstances = make(map[string]*MCPServerInstance)
	svc.temporaryMutex.Unlock()

	return nil
}

func (svc *service) RegisterMCPServer(ctx context.Context, id string, config MCPServerConfig, persistent ...bool) error {
	isPersistent := false
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	if id == "" {
		return ErrInvalidServerID
	}

	var instances map[string]*MCPServerInstance

	if isPersistent {
		instances = svc.persistentInstances
	} else {
		svc.temporaryMutex.Lock()
		defer svc.temporaryMutex.Unlock()

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
		c.Close()
		return err
	}

	instance := &MCPServerInstance{
		ID:     id,
		Client: c,
		Config: config,
	}

	instance.Beat()

	instances[id] = instance

	return nil
}

func (svc *service) UnregisterMCPServer(ctx context.Context, serverID string, persistent ...bool) error {
	isPersistent := false
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	if isPersistent {
		return ErrUnsupportedPersistentServerRemoval
	}

	if serverID == "" {
		return ErrInvalidServerID
	}

	svc.temporaryMutex.Lock()
	defer svc.temporaryMutex.Unlock()

	instance, ok := svc.temporaryInstances[serverID]
	if !ok {
		return ErrServerNotFound
	}

	delete(svc.temporaryInstances, serverID)

	return instance.Client.Close()
}

func (svc *service) healthMonitor(ctx context.Context, interval time.Duration) {
	log := svc.log.With(
		zap.String("action", "health_monitor"),
		zap.Duration("interval", interval),
	)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("done")
			return

		case <-ticker.C:
			log.Info("checking MCP server health")

			for id, instance := range svc.persistentInstances {
				log := log.With(
					zap.String("server_id", id),
					zap.String("type", "persistent"),
				)

				if err := instance.Client.Ping(ctx); err != nil {
					log.Error(err.Error())
					continue
				}

				instance.Beat()
				log.Info("server is alive")
			}

			svc.temporaryMutex.RLock()
			for id, instance := range svc.temporaryInstances {
				log := log.With(
					zap.String("server_id", id),
					zap.String("type", "temporary"),
				)

				if err := instance.Client.Ping(ctx); err != nil {
					log.Error(err.Error())
					continue
				}

				instance.Beat()
				log.Info("server is alive")
			}
			svc.temporaryMutex.RUnlock()
		}
	}
}

func (svc *service) cacheTools(ctx context.Context) {
	log := svc.log.With(
		zap.String("action", "refresh_tools_cache"),
	)

	var (
		routes = make(map[string]string)
		tools  = make([]mcp.Tool, 0)
	)

	for id, instance := range svc.persistentInstances {
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

			results, err := instance.Client.ListTools(ctx, req)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			instance.Beat()

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

				// Add tool to the vector database collection
				if svc.collection != nil {
					doc := ToolToDocument(tool, id)
					existingDoc, err := svc.collection.FindDocument(ctx, doc.ID)
					if err != nil || existingDoc.ID != doc.ID {
						err := svc.collection.AddDocument(ctx, doc)
						if err != nil {
							log.Error(err.Error())
							continue
						}

						log.Info("added tool document to vector collection")
					}
				}
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

	svc.toolRoutes = routes
	svc.toolsCache = tools

	log.Info("tools cached", zap.Int("count", len(tools)))
}

func (svc *service) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	serverID, ok := ctx.Value(ServerID).(string)
	if !ok {
		if len(svc.toolsCache) == 0 {
			return nil, ErrNoToolsFound
		}

		tools := make([]mcp.Tool, len(svc.toolsCache))
		copy(tools, svc.toolsCache)

		return tools, nil
	}

	svc.temporaryMutex.RLock()
	defer svc.temporaryMutex.RUnlock()

	instance, ok := svc.temporaryInstances[serverID]
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

		results, err := instance.Client.ListTools(ctx, req)
		if err != nil {
			return nil, err
		}

		instance.Beat()

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

func (svc *service) SearchTools(ctx context.Context, query string, k ...int) ([]mcp.Tool, error) {
	if svc.collection == nil {
		return nil, ErrVectorDBNotSet
	}

	n := 5 // Default number of results to return
	if len(k) > 0 && k[0] > 0 {
		n = k[0]
	}

	docs, err := svc.collection.Query(ctx, query, n)
	if err != nil {
		return nil, err
	}

	if len(docs) == 0 {
		return nil, ErrNoToolsFound
	}

	tools := make([]mcp.Tool, len(docs))
	for i, doc := range docs {
		toolJSON, ok := doc.Metadata["tool_json"]
		if !ok {
			return nil, ErrInvalidToolDocument
		}

		var tool mcp.Tool
		if err := json.Unmarshal([]byte(toolJSON), &tool); err != nil {
			return nil, err
		}

		tools[i] = tool
	}

	return tools, nil
}

func (svc *service) Forward(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	toolName := req.Params.Name

	serverID, ok := ctx.Value(ServerID).(string)
	if !ok {
		id, ok := svc.toolRoutes[toolName]
		if !ok {
			return nil, ErrToolNotFound
		}

		instance, ok := svc.persistentInstances[id]
		if !ok {
			return nil, ErrToolNotFound
		}

		if name, ok := strings.CutPrefix(toolName, id+":"); ok {
			req.Params.Name = name
		}

		result, err := instance.Client.CallTool(ctx, req)
		if err != nil {
			return nil, err
		}

		instance.Beat()

		return result, nil
	}

	svc.temporaryMutex.RLock()
	defer svc.temporaryMutex.RUnlock()

	instance, ok := svc.temporaryInstances[serverID]
	if !ok {
		return nil, ErrToolNotFound
	}

	result, err := instance.Client.CallTool(ctx, req)
	if err != nil {
		return nil, err
	}

	instance.Beat()

	return result, nil
}
