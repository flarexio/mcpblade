package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"github.com/flarexio/mcpblade"
	"github.com/flarexio/mcpblade/persistence/chromem"

	mcpE "github.com/flarexio/mcpblade/mcp"
	httpT "github.com/flarexio/mcpblade/transport/http"
	natsT "github.com/flarexio/mcpblade/transport/nats"
)

func main() {
	cmd := &cli.Command{
		Name:  "mcpblade",
		Usage: "MCPBlade service",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "Path to the MCPBlade service",
			},
			&cli.StringFlag{
				Name:    "nats",
				Usage:   "NATS server URL",
				Value:   "wss://nats.flarex.io",
				Sources: cli.EnvVars("NATS_URL"),
			},
			&cli.BoolFlag{
				Name:  "http",
				Usage: "Enable HTTP transport",
				Value: false,
			},
			&cli.StringFlag{
				Name:  "http-addr",
				Usage: "HTTP server address",
				Value: ":8080",
			},
		},
		Action: run,
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	path := cmd.String("path")
	if path == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		path = filepath.Join(homeDir, ".flarex", "mcpblade")
	}

	log, err := zap.NewDevelopment()
	if err != nil {
		return err
	}
	defer log.Sync()

	zap.ReplaceGlobals(log)

	f, err := os.Open(filepath.Join(path, "config.yaml"))
	if err != nil {
		return err
	}
	defer f.Close()

	var cfg mcpblade.Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return err
	}

	cfg.Vector.Path = filepath.Join(path, "vectors")

	vector, err := chromem.NewChromemVectorDB(cfg.Vector)
	if err != nil {
		return err
	}

	svc, err := mcpblade.NewService(ctx, cfg, vector)
	if err != nil {
		return err
	}
	defer svc.Close()

	svc = mcpblade.LoggingMiddleware(log)(svc)

	endpoints := mcpblade.EndpointSet{
		RegisterMCPServer:   mcpblade.RegisterMCPServerEndpoint(svc),
		UnregisterMCPServer: mcpblade.UnregisterMCPServerEndpoint(svc),
		ListTools:           mcpblade.ListToolsEndpoint(svc),
		SearchTools:         mcpblade.SearchToolsEndpoint(svc),
		Forward:             mcpblade.ForwardEndpoint(svc),
	}

	natsURL := cmd.String("nats")
	natsCreds := filepath.Join(path, "user.creds")

	idBytes, err := os.ReadFile(filepath.Join(path, "id"))
	if err != nil {
		return err
	}

	// Add NATS Transport
	{
		edgeID := strings.TrimSpace(string(idBytes))

		nc, err := nats.Connect(natsURL,
			nats.Name("MCPBlade Server - "+edgeID),
			nats.UserCredentials(natsCreds),
		)

		if err != nil {
			return err
		}
		defer nc.Drain()

		srv, err := micro.AddService(nc, micro.Config{
			Name:    "mcpblade",
			Version: "1.0.0",
		})

		if err != nil {
			return err
		}
		defer srv.Stop()

		topic := "edges." + edgeID + ".mcpblade"

		root := srv.AddGroup(topic)
		natsT.AddEndpoints(root, endpoints)
	}

	httpEnabled := cmd.Bool("http")
	if httpEnabled {
		r := gin.Default()
		httpT.AddRouters(r, endpoints)

		endpoints := make(map[mcp.MCPMethod]mcpE.MCPEndpoint)
		endpoints[mcp.MethodInitialize] = mcpE.InitializeEndpoint(svc)
		endpoints[mcp.MethodPing] = mcpE.PingEndpoint(svc)
		endpoints[mcp.MethodToolsList] = mcpE.ListToolsEndpoint(svc)
		endpoints[mcp.MethodToolsCall] = mcpE.CallToolEndpoint(svc)
		httpT.AddStreamableRouters(r, endpoints)

		httpAddr := cmd.String("http-addr")
		go r.Run(httpAddr)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	sign := <-quit

	log.Info("graceful shutdown", zap.String("signal", sign.String()))
	return nil
}
