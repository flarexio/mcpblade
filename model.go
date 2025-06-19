package mcpblade

import (
	"encoding/json"
	"errors"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	ErrUnsupportedTransportType = errors.New("unsupported transport type")
	ErrInvalidServerID          = errors.New("invalid server ID")
	ErrServerAlreadyExists      = errors.New("server already exists")
	ErrServerNotFound           = errors.New("server not found")
	ErrNoToolsFound             = errors.New("no tools found")
)

type ContextKey string

const (
	ServerID ContextKey = "server_id"
)

type Config struct {
	MCPServers      map[string]MCPServerConfig `yaml:"mcpServers"`
	CacheRefreshTTL time.Duration              `yaml:"cacheRefreshTTL"`
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
