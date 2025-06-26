package mcpblade

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"go.uber.org/zap"
)

func LoggingMiddleware(log *zap.Logger) ServiceMiddleware {
	log = log.With(
		zap.String("service", "mcpblade"),
	)

	return func(next Service) Service {
		log.Info("service initialized")

		return &loggingMiddleware{
			log:  log,
			next: next,
		}
	}
}

type loggingMiddleware struct {
	log  *zap.Logger
	next Service
}

func (mw *loggingMiddleware) Close() error {
	log := mw.log.With(
		zap.String("action", "close"),
	)

	err := mw.next.Close()
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("service closed")
	return nil
}

func (mw *loggingMiddleware) RegisterMCPServer(ctx context.Context, id string, config MCPServerConfig, persistent ...bool) error {
	isPersistent := false
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	log := mw.log.With(
		zap.String("action", "register_mcp_server"),
		zap.String("server_id", id),
		zap.Bool("persistent", isPersistent),
	)

	err := mw.next.RegisterMCPServer(ctx, id, config, persistent...)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("mcp server registered")
	return nil
}

func (mw *loggingMiddleware) UnregisterMCPServer(ctx context.Context, id string, persistent ...bool) error {
	isPersistent := false
	if len(persistent) > 0 {
		isPersistent = persistent[0]
	}

	log := mw.log.With(
		zap.String("action", "unregister_mcp_server"),
		zap.String("server_id", id),
		zap.Bool("persistent", isPersistent),
	)

	err := mw.next.UnregisterMCPServer(ctx, id, persistent...)
	if err != nil {
		log.Error(err.Error())
		return err
	}

	log.Info("mcp server unregistered")
	return nil
}

func (mw *loggingMiddleware) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	log := mw.log.With(
		zap.String("action", "list_tools"),
	)

	serverID, ok := ctx.Value(ServerID).(string)
	if ok {
		log = log.With(
			zap.String("server_id", serverID),
		)
	}

	tools, err := mw.next.ListTools(ctx)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	log.Info("tools listed", zap.Int("count", len(tools)))
	return tools, nil
}

func (mw *loggingMiddleware) SearchTools(ctx context.Context, query string, k ...int) ([]mcp.Tool, error) {
	var n int
	if len(k) > 0 {
		n = k[0]
	}

	log := mw.log.With(
		zap.String("action", "search_tools"),
		zap.String("query", query),
	)

	if n > 0 {
		log = log.With(
			zap.Int("k", n),
		)
	}

	tools, err := mw.next.SearchTools(ctx, query, k...)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	log.Info("tools searched", zap.Int("count", len(tools)))
	return tools, nil
}

func (mw *loggingMiddleware) Forward(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	log := mw.log.With(
		zap.String("action", "forward"),
		zap.String("method", req.Params.Name),
	)

	serverID, ok := ctx.Value(ServerID).(string)
	if ok {
		log = log.With(
			zap.String("server_id", serverID),
		)
	}

	result, err := mw.next.Forward(ctx, req)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	log.Info("request forwarded")
	return result, nil
}
