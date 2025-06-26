package http

import (
	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/flarexio/mcpblade"

	mcpE "github.com/flarexio/mcpblade/mcp"
)

func AddRouters(r *gin.Engine, endpoints mcpblade.EndpointSet) {
	// RESTful API routes
	api := r.Group("/api")
	{
		api.POST("/mcp/register", RegisterMCPServerHandler(endpoints.RegisterMCPServer))
		api.DELETE("/mcp/unregister/:server_id", UnregisterMCPServerHandler(endpoints.UnregisterMCPServer))
		api.GET("/mcp/tools", ListToolsHandler(endpoints.ListTools))
		api.GET("/mcp/tools/search", SearchToolsHandler(endpoints.SearchTools))
		api.POST("/mcp/forward", ForwardHandler(endpoints.Forward))
	}
}

func AddStreamableRouters(r *gin.Engine, endpoints map[mcp.MCPMethod]mcpE.MCPEndpoint) {
	mcp := r.Group("/mcp")
	{
		mcp.POST("/", MCPStreamableHandler(endpoints))
		// mcp.GET("/sse", MCPSSSEHandler(endpoints))
	}
}
