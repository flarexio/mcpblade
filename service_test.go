package mcpblade

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type mcpBladeTestSuite struct {
	suite.Suite
	ctx context.Context
	svc Service
}

func (suite *mcpBladeTestSuite) SetupTest() {
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
		},
		CacheRefreshTTL: 5 * time.Minute,
	}

	svc := NewService(ctx, cfg)

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

	suite.Len(tools, 2)
	suite.Equal("get_current_time", tools[0].Name)
}

func TestMCPBladeTestSuite(t *testing.T) {
	suite.Run(t, new(mcpBladeTestSuite))
}
