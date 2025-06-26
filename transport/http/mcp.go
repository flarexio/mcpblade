package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"

	mcpE "github.com/flarexio/mcpblade/mcp"
)

func MCPStreamableHandler(endpoints map[mcp.MCPMethod]mcpE.MCPEndpoint) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req mcpE.JSONRPCRequest
		if err := c.ShouldBind(&req); err != nil {
			c.Error(err)
			c.Abort()

			resp := mcp.JSONRPCError{
				JSONRPC: mcp.JSONRPC_VERSION,
				ID:      req.ID,
				Error: struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Data    any    `json:"data,omitempty"`
				}{
					Code:    mcp.METHOD_NOT_FOUND,
					Message: "method not found",
				},
			}
			c.JSON(http.StatusNotFound, &resp)
			return
		}

		endpoint, ok := endpoints[req.Method]
		if !ok {
			err := errors.New("endpoint not found")
			c.Error(err)
			c.Abort()

			resp := mcp.JSONRPCError{
				JSONRPC: mcp.JSONRPC_VERSION,
				ID:      req.ID,
				Error: struct {
					Code    int    `json:"code"`
					Message string `json:"message"`
					Data    any    `json:"data,omitempty"`
				}{
					Code:    mcp.METHOD_NOT_FOUND,
					Message: "method not found",
				},
			}
			c.JSON(http.StatusNotFound, &resp)
			return
		}

		ctx := c.Request.Context()
		resp := endpoint(ctx, req)

		c.JSON(http.StatusOK, &resp)
	}
}
