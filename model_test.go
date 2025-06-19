package mcpblade

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func TestMCPServerConfigJSONUnmarshal(t *testing.T) {
	assert := assert.New(t)

	input := `{
		"transport": "stdio",
		"command": "uvx",
		"args": [ 
			"mcp-server-time", 
			"--local-timezone=Asia/Taipei" 
		]
	}`

	var config MCPServerConfig
	if err := json.Unmarshal([]byte(input), &config); err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal(TransportTypeStdio, config.Transport)
	assert.Equal("uvx", config.Command)
	assert.Equal(time.Duration(0), config.TTL.Duration(), "permanent server should have TTL 0")
}

func TestMCPServerConfigYAMLUnmarshal(t *testing.T) {
	assert := assert.New(t)

	input := `transport: stdio
command: uvx
args:
  - mcp-server-time
  - --local-timezone=Asia/Taipei`

	var config MCPServerConfig
	if err := yaml.Unmarshal([]byte(input), &config); err != nil {
		assert.Fail(err.Error())
		return
	}

	assert.Equal(TransportTypeStdio, config.Transport)
	assert.Equal("uvx", config.Command)
	assert.Equal(time.Duration(0), config.TTL.Duration(), "permanent server should have TTL")
}
