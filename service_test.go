package mcpblade

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/suite"

	"github.com/flarexio/mcpblade/persistence/chromem"
	"github.com/flarexio/mcpblade/vector"
)

type mcpBladeTestSuite struct {
	suite.Suite
	ctx context.Context
	svc Service
}

func (suite *mcpBladeTestSuite) SetupSuite() {
	ctx := context.Background()

	cfg := Config{
		MCPServers: map[string]MCPServerConfig{
			"time": {
				Transport: TransportTypeStdio,
				Command:   "uvx",
				Arguments: []string{
					"mcp-server-time",
					"--local-timezone=Asia/Taipei",
				},
			},
			"time2": {
				Transport: TransportTypeStdio,
				Command:   "uvx",
				Arguments: []string{
					"mcp-server-time",
					"--local-timezone=Asia/Taipei",
				},
			},
		},
		CacheRefreshTTL: 5 * time.Minute,
		Vector: vector.Config{
			Enabled:    true,
			Persistent: false,
			Collection: "tools",
		},
	}

	vector, err := chromem.NewChromemVectorDB(cfg.Vector)
	if err != nil {
		suite.Fail(err.Error())
		return
	}

	svc, err := NewService(ctx, cfg, vector)
	if err != nil {
		suite.Fail(err.Error())
		return
	}

	suite.ctx = ctx
	suite.svc = svc
}

func (suite *mcpBladeTestSuite) TestRegisterMCPServer() {
	ctx := context.Background()

	config := MCPServerConfig{
		Transport: TransportTypeStdio,
		Command:   "uvx",
		Arguments: []string{
			"mcp-server-time",
			"--local-timezone=Asia/Taipei",
		},
	}

	err := suite.svc.RegisterMCPServer(ctx, "test-time", config)
	suite.NoError(err)

	ctx = context.WithValue(ctx, ServerID, "test-time")

	tools, err := suite.svc.ListTools(ctx)
	if err != nil {
		suite.Fail(err.Error())
		return
	}

	suite.Len(tools, 2)
	suite.Equal("get_current_time", tools[0].Name)
}

func (suite *mcpBladeTestSuite) TestListTools() {
	ctx := context.Background()

	tools, err := suite.svc.ListTools(ctx)
	if err != nil {
		suite.Fail(err.Error())
		return
	}

	suite.Len(tools, 4)
}

func (suite *mcpBladeTestSuite) TestSearchTool() {
	ctx := context.Background()

	tools, err := suite.svc.SearchTools(ctx, "what's the time?")
	if err != nil {
		suite.Fail(err.Error())
		return
	}

	suite.Len(tools, 4)
	suite.Contains(tools[0].Name, "get_current_time")
	suite.Contains(tools[1].Name, "get_current_time")
}

func (suite *mcpBladeTestSuite) TestForward() {
	ctx := context.Background()

	req := mcp.CallToolRequest{
		Request: mcp.Request{
			Method: "tools/call",
		},
		Params: mcp.CallToolParams{
			Name: "get_current_time",
			Arguments: map[string]any{
				"timezone": "Asia/Taipei",
			},
		},
	}

	result, err := suite.svc.Forward(ctx, req)
	if err != nil {
		suite.Fail(err.Error())
		return
	}

	suite.False(result.IsError)
	suite.Len(result.Content, 1)

	content, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		suite.Fail("invalid type")
		return
	}

	var output struct {
		Timezone string    `json:"timezone"`
		DateTime time.Time `json:"datetime"`
		IsDST    bool      `json:"is_dst"`
	}

	if err := json.Unmarshal([]byte(content.Text), &output); err != nil {
		suite.Fail(err.Error())
		return
	}

	suite.Equal("Asia/Taipei", output.Timezone)
	suite.True(time.Since(output.DateTime) < 5*time.Second, "datetime should be recent")
	suite.False(output.IsDST)
}

func (suite *mcpBladeTestSuite) TearDownSuite() {
	if suite.svc != nil {
		suite.svc.Close()
	}

	suite.ctx = nil
	suite.svc = nil
}

func TestMCPBladeTestSuite(t *testing.T) {
	suite.Run(t, new(mcpBladeTestSuite))
}
