package nats

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/go-kit/kit/endpoint"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"

	"github.com/flarexio/mcpblade"
)

func MakeEndpoints(nc *nats.Conn, prefix string) *mcpblade.EndpointSet {
	return &mcpblade.EndpointSet{
		RegisterMCPServer:   RegisterMCPServerEndpoint(nc, prefix+".register_mcp_server"),
		UnregisterMCPServer: UnregisterMCPServerEndpoint(nc, prefix+".unregister_mcp_server"),
		ListTools:           ListToolsEndpoint(nc, prefix+".list_tools"),
		SearchTools:         SearchToolsEndpoint(nc, prefix+".search_tools"),
		Forward:             ForwardEndpoint(nc, prefix+".forward"),
	}
}

func RegisterMCPServerEndpoint(nc *nats.Conn, topic string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(mcpblade.RegisterMCPServerRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		data, err := json.Marshal(&req)
		if err != nil {
			return nil, err
		}

		resp, err := nc.Request(topic, data, nats.DefaultTimeout)
		if err != nil {
			return nil, err
		}

		return string(resp.Data), nil
	}
}

func UnregisterMCPServerEndpoint(nc *nats.Conn, topic string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		serverID, ok := request.(string)
		if !ok {
			return nil, errors.New("invalid request")
		}

		resp, err := nc.Request(topic, []byte(serverID), nats.DefaultTimeout)
		if err != nil {
			return nil, err
		}

		return string(resp.Data), nil
	}
}

func ListToolsEndpoint(nc *nats.Conn, topic string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		header := make(nats.Header)

		serverID, ok := ctx.Value(mcpblade.ServerID).(string)
		if ok {
			header.Set("server_id", serverID)
		}

		msg := nats.NewMsg(topic)
		msg.Header = header
		msg.Data = nil

		resp, err := nc.RequestMsg(msg, nats.DefaultTimeout)
		if err != nil {
			return nil, err
		}

		var tools []mcp.Tool
		if err := json.Unmarshal(resp.Data, &tools); err != nil {
			return nil, err
		}

		return tools, nil
	}
}

func SearchToolsEndpoint(nc *nats.Conn, topic string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(mcpblade.SearchToolsRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		data, err := json.Marshal(&req)
		if err != nil {
			return nil, err
		}

		resp, err := nc.Request(topic, data, nats.DefaultTimeout)
		if err != nil {
			return nil, err
		}

		var tools []mcp.Tool
		if err := json.Unmarshal(resp.Data, &tools); err != nil {
			return nil, err
		}

		return tools, nil
	}
}

func ForwardEndpoint(nc *nats.Conn, topic string) endpoint.Endpoint {
	return func(ctx context.Context, request any) (any, error) {
		req, ok := request.(mcp.CallToolRequest)
		if !ok {
			return nil, errors.New("invalid request")
		}

		data, err := json.Marshal(&req)
		if err != nil {
			return nil, err
		}

		header := make(nats.Header)

		serverID, ok := ctx.Value(mcpblade.ServerID).(string)
		if ok {
			header.Set("server_id", serverID)
		}

		msg := nats.NewMsg(topic)
		msg.Header = header
		msg.Data = data

		resp, err := nc.RequestMsg(msg, nats.DefaultTimeout)
		if err != nil {
			return nil, err
		}

		raw := json.RawMessage(resp.Data)

		return mcp.ParseCallToolResult(&raw)
	}
}

func Error(msg *nats.Msg) error {
	if msg == nil {
		return errors.New("nil message")
	}

	code := msg.Header.Get(micro.ErrorCodeHeader)
	if code == "" {
		return nil
	}

	description := msg.Header.Get(micro.ErrorHeader)
	if description == "" {
		description = "unknown error"
	}

	return errors.New(code + ":" + description)
}
