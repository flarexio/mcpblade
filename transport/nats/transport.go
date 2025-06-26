package nats

import (
	"context"
	"encoding/json"

	"github.com/go-kit/kit/endpoint"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-io/nats.go/micro"

	"github.com/flarexio/mcpblade"
)

func RegisterMCPServerHandler(endpoint endpoint.Endpoint) micro.HandlerFunc {
	return func(r micro.Request) {
		var req mcpblade.RegisterMCPServerRequest
		if err := json.Unmarshal(r.Data(), &req); err != nil {
			r.Error("400", err.Error(), nil)
			return
		}

		ctx := context.Background()
		_, err := endpoint(ctx, req)
		if err != nil {
			r.Error("417", err.Error(), nil)
			return
		}

		r.Respond([]byte("OK"))
	}
}

func UnregisterMCPServerHandler(endpoint endpoint.Endpoint) micro.HandlerFunc {
	return func(r micro.Request) {
		serverID := string(r.Data())
		if serverID == "" {
			r.Error("400", "server id is required", nil)
			return
		}

		ctx := context.Background()
		_, err := endpoint(ctx, serverID)
		if err != nil {
			r.Error("417", err.Error(), nil)
			return
		}

		r.Respond([]byte("OK"))
	}
}

func ListToolsHandler(endpoint endpoint.Endpoint) micro.HandlerFunc {
	return func(r micro.Request) {
		ctx := context.Background()

		serverID := r.Headers().Get("server_id")
		if serverID != "" {
			ctx = context.WithValue(ctx, mcpblade.ServerID, serverID)
		}

		resp, err := endpoint(ctx, nil)
		if err != nil {
			r.Error("417", err.Error(), nil)
			return
		}

		tools, ok := resp.([]mcp.Tool)
		if !ok {
			r.Error("500", "invalid response type", nil)
			return
		}

		r.RespondJSON(&tools)
	}
}

func SearchToolsHandler(endpoint endpoint.Endpoint) micro.HandlerFunc {
	return func(r micro.Request) {
		var req mcpblade.SearchToolsRequest
		if err := json.Unmarshal(r.Data(), &req); err != nil {
			r.Error("400", err.Error(), nil)
			return
		}

		ctx := context.Background()
		resp, err := endpoint(ctx, req)
		if err != nil {
			r.Error("417", err.Error(), nil)
			return
		}

		tools, ok := resp.([]mcp.Tool)
		if !ok {
			r.Error("500", "invalid response type", nil)
			return
		}

		r.RespondJSON(&tools)
	}
}

func ForwardHandler(endpoint endpoint.Endpoint) micro.HandlerFunc {
	return func(r micro.Request) {
		var req mcpblade.ForwardRequest
		if err := json.Unmarshal(r.Data(), &req); err != nil {
			r.Error("400", err.Error(), nil)
			return
		}

		ctx := context.Background()

		serverID := r.Headers().Get("server_id")
		if serverID != "" {
			ctx = context.WithValue(ctx, mcpblade.ServerID, serverID)
		}

		resp, err := endpoint(ctx, req)
		if err != nil {
			r.Error("417", err.Error(), nil)
			return
		}

		r.RespondJSON(&resp)
	}
}
