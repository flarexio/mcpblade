package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/nats-io/nats.go"
	"github.com/urfave/cli/v3"

	"github.com/flarexio/mcpblade"

	mcpE "github.com/flarexio/mcpblade/mcp"
	natsT "github.com/flarexio/mcpblade/transport/nats"
)

type StdioMCPServer interface {
	AddEndpoint(method mcp.MCPMethod, endpoint mcpE.MCPEndpoint) error
	Listen(ctx context.Context) error
}

func NewStdioMCPServer() StdioMCPServer {
	return &stdioMCPServer{
		endpoints: make(map[mcp.MCPMethod]mcpE.MCPEndpoint),
	}
}

type stdioMCPServer struct {
	endpoints map[mcp.MCPMethod]mcpE.MCPEndpoint
}

func (s *stdioMCPServer) Listen(ctx context.Context) error {
	scanner := bufio.NewScanner(os.Stdin)

	lines := make(chan string)
	errs := make(chan error, 1)

	go func(ctx context.Context, lines chan<- string, errs chan<- error) {
		defer close(lines)

		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-ctx.Done():
				return
			}
		}

		if err := scanner.Err(); err != nil {
			errs <- err
		}
	}(ctx, lines, errs)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case err := <-errs:
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err

		case line, ok := <-lines:
			if !ok {
				return nil
			}

			if line == "" {
				continue
			}

			var req mcpE.JSONRPCRequest
			if err := json.Unmarshal([]byte(line), &req); err != nil {
				continue
			}

			if req.ID.IsNil() {
				continue
			}

			var resp mcp.JSONRPCMessage

			endpoint, ok := s.endpoints[req.Method]
			if !ok {
				resp = mcp.JSONRPCError{
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
			}

			resp = endpoint(ctx, req)

			bs, err := json.Marshal(resp)
			if err != nil {
				continue
			}

			fmt.Fprintf(os.Stdout, "%s\n", bs)
		}
	}
}

func (srv *stdioMCPServer) AddEndpoint(method mcp.MCPMethod, endpoint mcpE.MCPEndpoint) error {
	_, ok := srv.endpoints[method]
	if ok {
		return errors.New("endpoint already exists")
	}

	srv.endpoints[method] = endpoint
	return nil
}

func main() {
	cmd := &cli.Command{
		Name:  "mcpblade_mcp_server",
		Usage: "MCPBlade MCP Server",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "nats",
				Usage:   "NATS server URL",
				Value:   "wss://nats.flarex.io",
				Sources: cli.EnvVars("NATS_URL"),
			},
			&cli.StringFlag{
				Name:    "nats-creds",
				Usage:   "NATS user credentials file",
				Sources: cli.EnvVars("NATS_CREDS"),
			},
			&cli.StringFlag{
				Name:     "edge-id",
				Usage:    "Edge ID for connecting to the MCPBlade service",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "server-id",
				Usage: "Server ID for dedicated 1:1 connection. If not specified, uses shared 1:n resources",
			},
			&cli.StringFlag{
				Name:  "cmd",
				Usage: "Command to run for the MCP server",
			},
		},
		ArgsUsage: "[command and arguments...]",
		Action:    run,
	}

	err := cmd.Run(context.Background(), os.Args)
	if err != nil {
		log.Fatal(err.Error())
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	edgeID := cmd.String("edge-id")
	natsURL := cmd.String("nats")
	natsCreds := cmd.String("nats-creds")

	nc, err := nats.Connect(natsURL,
		nats.Name("MCPBlade MCP Server - "+edgeID),
		nats.UserCredentials(natsCreds),
	)

	if err != nil {
		log.Fatal(err.Error())
	}
	defer nc.Drain()

	topic := fmt.Sprintf("edges.%s.mcpblade", edgeID)
	endpoints := natsT.MakeEndpoints(nc, topic)

	var svc mcpblade.Service
	svc = mcpblade.ProxyMiddleware(endpoints)(svc)

	serverID := cmd.String("server-id")
	if serverID != "" {
		ctx = context.WithValue(ctx, mcpblade.ServerID, serverID)

		cmdLine := cmd.String("cmd")
		if cmdLine == "" {
			return errors.New("server-id specified but no command provided")
		}

		commandArgs := strings.Fields(cmdLine)
		if len(commandArgs) == 0 {
			return errors.New("no command provided for MCP server")
		}

		cfg := mcpblade.MCPServerConfig{
			Transport: mcpblade.TransportTypeStdio,
			Command:   commandArgs[0],
		}

		if len(commandArgs) > 1 {
			cfg.Arguments = commandArgs[1:]
		}

		err := svc.RegisterMCPServer(ctx, serverID, cfg)
		if err != nil {
			return err
		}

		defer svc.UnregisterMCPServer(ctx, serverID)
	}

	s := NewStdioMCPServer()
	s.AddEndpoint(mcp.MethodInitialize, mcpE.InitializeEndpoint(svc))
	s.AddEndpoint(mcp.MethodPing, mcpE.PingEndpoint(svc))
	s.AddEndpoint(mcp.MethodToolsList, mcpE.ListToolsEndpoint(svc))
	s.AddEndpoint(mcp.MethodToolsCall, mcpE.CallToolEndpoint(svc))

	go s.Listen(ctx)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	<-quit

	cancel()
	return nil
}
