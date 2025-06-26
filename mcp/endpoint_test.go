package mcp

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalInitializeRequest(t *testing.T) {
	assert := assert.New(t)

	input := []byte(`{
	  "jsonrpc": "2.0",
	  "id": 1,
	  "method": "initialize",
	  "params": {
	    "protocolVersion": "2024-11-05",
	    "capabilities": {
	      "roots": {
	        "listChanged": true
	      },
	      "sampling": {},
	      "elicitation": {}
	    },
	    "clientInfo": {
	      "name": "ExampleClient",
	      "title": "Example Client Display Name",
	      "version": "1.0.0"
	    }
	  }
	}`)

	var req JSONRPCRequest
	if err := json.Unmarshal(input, &req); err != nil {
		assert.Fail(err.Error())
		return
	}

	var params mcp.InitializeParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal(mcp.JSONRPC_VERSION, req.JSONRPC)
	assert.Equal(mcp.NewRequestId(int64(1)), req.ID)
	assert.Equal(mcp.MethodInitialize, req.Method)
	assert.Equal("2024-11-05", params.ProtocolVersion)
}

func TestUnmarshalCallToolRequest(t *testing.T) {
	assert := assert.New(t)

	input := []byte(`{
	  "jsonrpc": "2.0",
	  "id": 2,
	  "method": "tools/call",
	  "params": {
	    "name": "get_weather",
	    "arguments": {
	      "location": "New York"
	    }
	  }
	}`)

	var req JSONRPCRequest
	if err := json.Unmarshal(input, &req); err != nil {
		assert.Fail(err.Error())
		return
	}

	var params mcp.CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal(mcp.JSONRPC_VERSION, req.JSONRPC)
	assert.Equal(mcp.NewRequestId(int64(2)), req.ID)
	assert.Equal(mcp.MethodToolsCall, req.Method)
	assert.Equal("get_weather", params.Name)
	assert.Contains(params.Arguments, "location")

	var callToolReq mcp.CallToolRequest
	if err := json.Unmarshal(input, &callToolReq); err != nil {
		assert.Fail(err.Error())
		return
	}
}
