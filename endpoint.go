package mcpblade

import (
	"context"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/mark3labs/mcp-go/mcp"
)

type EndpointSet struct {
	RegisterMCPServer   endpoint.Endpoint
	UnregisterMCPServer endpoint.Endpoint
	ListTools           endpoint.Endpoint
	SearchTools         endpoint.Endpoint
	Forward             endpoint.Endpoint
}

type RegisterMCPServerRequest struct {
	ServerID   string          `json:"server_id"`
	Config     MCPServerConfig `json:"config"`
	Persistent bool            `json:"persistent,omitempty"`
}

func RegisterMCPServerEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(RegisterMCPServerRequest)
		if !ok {
			return nil, errors.New("invalid request type")
		}

		if req.Persistent {
			return nil, errors.New("persistent registration is not supported")
		}

		err := svc.RegisterMCPServer(ctx, req.ServerID, req.Config)
		return nil, err
	}
}

func UnregisterMCPServerEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		serverID, ok := request.(string)
		if !ok {
			return nil, errors.New("invalid request type")
		}

		err := svc.UnregisterMCPServer(ctx, serverID)
		return nil, err
	}
}

func ListToolsEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		return svc.ListTools(ctx)
	}
}

type SearchToolsRequest struct {
	Query string `json:"query"`
	K     int    `json:"k,omitempty"`
}

func SearchToolsEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(SearchToolsRequest)
		if !ok {
			return nil, errors.New("invalid request type")
		}

		return svc.SearchTools(ctx, req.Query, req.K)
	}
}

type ForwardRequest = mcp.CallToolRequest

func ForwardEndpoint(svc Service) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(ForwardRequest)
		if !ok {
			return nil, errors.New("invalid request type")
		}

		resp, err := svc.Forward(ctx, req)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}
}
