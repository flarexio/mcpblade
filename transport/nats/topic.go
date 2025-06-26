package nats

import (
	"github.com/nats-io/nats.go/micro"

	"github.com/flarexio/mcpblade"
)

func AddEndpoints(group micro.Group, endpoints mcpblade.EndpointSet) {
	group.AddEndpoint("register_mcp_server", RegisterMCPServerHandler(endpoints.RegisterMCPServer))
	group.AddEndpoint("unregister_mcp_server", UnregisterMCPServerHandler(endpoints.UnregisterMCPServer))
	group.AddEndpoint("list_tools", ListToolsHandler(endpoints.ListTools))
	group.AddEndpoint("search_tools", SearchToolsHandler(endpoints.SearchTools))
	group.AddEndpoint("forward", ForwardHandler(endpoints.Forward))
}
