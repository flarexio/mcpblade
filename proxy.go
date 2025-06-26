package mcpblade

import (
	"context"
	"errors"

	"github.com/mark3labs/mcp-go/mcp"
)

func ProxyMiddleware(endpoints *EndpointSet) ServiceMiddleware {
	return func(next Service) Service {
		return &proxyMiddleware{
			endpoints: endpoints,
		}
	}
}

type proxyMiddleware struct {
	endpoints *EndpointSet
}

func (mw *proxyMiddleware) Close() error {
	return errors.New("method not implemented")
}

func (mw *proxyMiddleware) RegisterMCPServer(ctx context.Context, id string, config MCPServerConfig, persistent ...bool) error {
	if len(persistent) > 0 && persistent[0] {
		return errors.New("persistent servers are not supported")
	}

	req := RegisterMCPServerRequest{
		ServerID: id,
		Config:   config,
	}

	_, err := mw.endpoints.RegisterMCPServer(ctx, req)
	return err
}

func (mw *proxyMiddleware) UnregisterMCPServer(ctx context.Context, id string, persistent ...bool) error {
	if len(persistent) > 0 && persistent[0] {
		return errors.New("persistent servers are not supported")
	}

	_, err := mw.endpoints.UnregisterMCPServer(ctx, id)
	return err
}

func (mw *proxyMiddleware) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	resp, err := mw.endpoints.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}

	tools, ok := resp.([]mcp.Tool)
	if !ok {
		return nil, errors.New("invalid response type")
	}

	return tools, nil
}

func (mw *proxyMiddleware) SearchTools(ctx context.Context, query string, k ...int) ([]mcp.Tool, error) {
	n := 0
	if len(k) > 0 {
		n = k[0]
	}

	req := SearchToolsRequest{
		Query: query,
		K:     n,
	}

	resp, err := mw.endpoints.SearchTools(ctx, req)
	if err != nil {
		return nil, err
	}

	tools, ok := resp.([]mcp.Tool)
	if !ok {
		return nil, errors.New("invalid response type")
	}

	return tools, nil
}

func (mw *proxyMiddleware) Forward(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := mw.endpoints.Forward(ctx, req)
	if err != nil {
		return nil, err
	}

	result, ok := resp.(*mcp.CallToolResult)
	if !ok {
		return nil, errors.New("invalid response type")
	}

	return result, nil
}
