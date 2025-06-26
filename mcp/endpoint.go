package mcp

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/flarexio/mcpblade"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      mcp.RequestId   `json:"id"`
	Method  mcp.MCPMethod   `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

func errorResponse(id any, code int, message string) mcp.JSONRPCError {
	return mcp.JSONRPCError{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(id),
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    any    `json:"data,omitempty"`
		}{
			Code:    code,
			Message: message,
		},
	}
}

type MCPEndpoint func(ctx context.Context, req JSONRPCRequest) mcp.JSONRPCMessage

const MCPSERVER_INSTRUCTIONS string = `MCPBlade aggregates and manages tools from multiple MCP servers, providing:

1. **Tool Discovery**: List all available tools from connected servers
2. **Semantic Search**: Find tools using natural language queries  
3. **Smart Routing**: Automatically routes tool calls to the correct backend server
4. **Vector Search**: Tools are indexed for intelligent search capabilities

Available operations:
- tools/list: Get all available tools
- tools/call: Execute tools (automatically routed)
- search_tools: Find tools using semantic search

All tools are enhanced with server information and deduplicated for easy discovery.`

func InitializeEndpoint(svc mcpblade.Service) MCPEndpoint {
	return func(ctx context.Context, req JSONRPCRequest) mcp.JSONRPCMessage {
		var params mcp.InitializeParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, mcp.INVALID_PARAMS, err.Error())
		}

		protocolVersion := mcp.LATEST_PROTOCOL_VERSION
		if clientVersion := params.ProtocolVersion; clientVersion != "" {
			if slices.Contains(mcp.ValidProtocolVersions, clientVersion) {
				protocolVersion = clientVersion
			}
		}

		result := &mcp.InitializeResult{
			ProtocolVersion: protocolVersion,
			Capabilities: mcp.ServerCapabilities{
				Tools: &struct {
					ListChanged bool `json:"listChanged,omitempty"`
				}{},
			},
			ServerInfo: mcp.Implementation{
				Name:    "mcpblade",
				Version: "1.0.0",
			},
			Instructions: MCPSERVER_INSTRUCTIONS,
		}

		return mcp.JSONRPCResponse{
			JSONRPC: mcp.JSONRPC_VERSION,
			ID:      req.ID,
			Result:  result,
		}
	}
}

func PingEndpoint(svc mcpblade.Service) MCPEndpoint {
	return func(ctx context.Context, req JSONRPCRequest) mcp.JSONRPCMessage {
		return mcp.JSONRPCResponse{
			JSONRPC: mcp.JSONRPC_VERSION,
			ID:      req.ID,
			Result:  struct{}{}, // empty response
		}
	}
}

func ListToolsEndpoint(svc mcpblade.Service) MCPEndpoint {
	return func(ctx context.Context, req JSONRPCRequest) mcp.JSONRPCMessage {
		tools, err := svc.ListTools(ctx)
		if err != nil {
			return errorResponse(req.ID, mcp.INTERNAL_ERROR, err.Error())
		}

		result := &mcp.ListToolsResult{
			Tools: tools,
		}

		return mcp.JSONRPCResponse{
			JSONRPC: mcp.JSONRPC_VERSION,
			ID:      req.ID,
			Result:  result,
		}
	}
}

func CallToolEndpoint(svc mcpblade.Service) MCPEndpoint {
	return func(ctx context.Context, req JSONRPCRequest) mcp.JSONRPCMessage {
		var params mcp.CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, mcp.INVALID_PARAMS, err.Error())
		}

		callToolReq := mcp.CallToolRequest{
			Request: mcp.Request{
				Method: string(req.Method),
			},
			Params: params,
		}

		result, err := svc.Forward(ctx, callToolReq)
		if err != nil {
			return errorResponse(req.ID, mcp.INTERNAL_ERROR, err.Error())
		}

		return mcp.JSONRPCResponse{
			JSONRPC: mcp.JSONRPC_VERSION,
			ID:      req.ID,
			Result:  result,
		}
	}
}
