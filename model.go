package mcpblade

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/mcpblade/vector"
)

var (
	ErrUnsupportedTransportType           = errors.New("unsupported transport type")
	ErrInvalidServerID                    = errors.New("invalid server ID")
	ErrServerAlreadyExists                = errors.New("server already exists")
	ErrServerNotFound                     = errors.New("server not found")
	ErrNoToolsFound                       = errors.New("no tools found")
	ErrToolNotFound                       = errors.New("tool not found")
	ErrVectorDBNotSet                     = errors.New("vector database not set")
	ErrInvalidToolDocument                = errors.New("invalid tool document")
	ErrUnsupportedPersistentServerRemoval = errors.New("removal of persistent servers is not supported")
)

type ContextKey string

const (
	EdgeID   ContextKey = "edge_id"
	ServerID ContextKey = "server_id"
)

type Config struct {
	MCPServers      map[string]MCPServerConfig `yaml:"mcpServers"`
	CacheRefreshTTL time.Duration              `yaml:"cacheRefreshTTL"`
	Vector          vector.Config              `yaml:"vector"`
}

type TransportType string

const (
	TransportTypeStdio          TransportType = "stdio"
	TransportTypeSSE            TransportType = "sse"
	TransportTypeStreamableHTTP TransportType = "streamable-http"
	TransportTypeNATS           TransportType = "nats"
)

type RestartPolicy string

const (
	RestartPolicyAlways    RestartPolicy = "always"
	RestartPolicyOnFailure RestartPolicy = "on-failure"
	RestartPolicyNever     RestartPolicy = "never"
)

type Duration time.Duration

func (d Duration) Duration() time.Duration {
	return time.Duration(d)
}

func (d Duration) MarshalJSON() ([]byte, error) {
	str := d.Duration().String()
	return json.Marshal(str)
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	// Parse the string duration
	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

func (d Duration) MarshalYAML() ([]byte, error) {
	str := d.Duration().String()
	return yaml.Marshal(str)
}

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var str string
	if err := value.Decode(&str); err != nil {
		return err
	}

	// Parse the string duration
	duration, err := time.ParseDuration(str)
	if err != nil {
		return err
	}

	*d = Duration(duration)
	return nil
}

type MCPServerConfig struct {
	Transport     TransportType `json:"transport" yaml:"transport"`
	Command       string        `json:"command" yaml:"command"`
	URL           string        `json:"url" yaml:"url"`
	Arguments     []string      `json:"args" yaml:"args"`
	Environment   []string      `json:"env" yaml:"env"`
	RestartPolicy RestartPolicy `json:"restart" yaml:"restart"`
	TTL           Duration      `json:"ttl" yaml:"ttl"`
}

type MCPServerInstance struct {
	ID     string
	Client client.MCPClient
	Config MCPServerConfig

	heartbeat atomic.Int64
}

func (i *MCPServerInstance) Beat() {
	i.heartbeat.Store(time.Now().UnixNano())
}

func (i *MCPServerInstance) IsAlive() bool {
	if i.heartbeat.Load() == 0 {
		return false
	}

	lastBeat := time.Unix(0, i.heartbeat.Load())
	return time.Since(lastBeat) < i.Config.TTL.Duration()
}

func ToolToDocument(tool mcp.Tool, serverID string) vector.Document {
	return vector.Document{
		ID:       generateDocumentID(tool, serverID),
		Content:  buildSearchContent(tool),
		Metadata: buildMetadata(tool, serverID),
	}
}

func generateDocumentID(tool mcp.Tool, serverID string) string {
	data := fmt.Sprintf("%s|%s|%s", serverID, tool.Name, tool.Description)

	bs, err := json.Marshal(tool.InputSchema)
	if err == nil {
		data += "|" + string(bs)
	}

	hash := sha256.Sum256([]byte(data))
	return "tool_" + hex.EncodeToString(hash[:12])
}

func buildSearchContent(tool mcp.Tool) string {
	var parts []string

	parts = append(parts, tool.Name)

	if tool.Description != "" {
		parts = append(parts, tool.Description)
	}

	if tool.Annotations.Title != "" {
		parts = append(parts, tool.Annotations.Title)
	}

	return strings.Join(parts, " ")
}

func buildMetadata(tool mcp.Tool, serverID string) map[string]string {
	metadata := map[string]string{
		"server_id":   serverID,
		"tool_name":   tool.Name,
		"description": tool.Description,
	}

	if bs, err := json.Marshal(tool); err == nil {
		metadata["tool_json"] = string(bs)
	}

	return metadata
}
