// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/mcpserver"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	defaultTimeout = 30 * time.Second
)

func main() {
	// Parse command line flags
	var (
		serverURL    = flag.String("server-url", "", "Mattermost server URL (required)")
		token        = flag.String("token", "", "Personal Access Token")
		transport    = flag.String("transport", "stdio", "Transport type (stdio, http) - stdio is default")
		httpPort     = flag.Int("port", 8080, "HTTP port for http transport (default: 8080)")
		debug        = flag.Bool("debug", false, "Enable debug logging")
		logFile      = flag.String("logfile", "", "Path to log file (logs to file in addition to stderr)")
		clientID     = flag.String("oauth-client-id", "", "OAuth client ID (required for OAuth auth)")
		clientSecret = flag.String("oauth-client-secret", "", "OAuth client secret (required for OAuth auth)")
		redirectURL  = flag.String("oauth-redirect-url", "", "OAuth redirect URL (required for OAuth auth)")
		devMode      = flag.Bool("dev", false, "Enable development mode with additional tools for setting up test data")
		version      = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Fprintf(os.Stderr, "Mattermost MCP Server v0.1.0\n")
		os.Exit(0)
	}

	// Set up mlog logger
	logger, err := mlog.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}

	cfg := make(mlog.LoggerConfiguration)
	cfg["console"] = mlog.TargetCfg{
		Type:          "console",
		Levels:        mlog.StdAll,
		Format:        "plain",
		FormatOptions: json.RawMessage(`{"enable_color": false}`),
		Options:       json.RawMessage(`{"out": "stderr"}`),
		MaxQueueSize:  1000,
	}

	if *debug {
		cfg["console"] = mlog.TargetCfg{
			Type:          "console",
			Levels:        []mlog.Level{mlog.LvlDebug, mlog.LvlInfo, mlog.LvlWarn, mlog.LvlError},
			Format:        "plain",
			FormatOptions: json.RawMessage(`{"enable_color": false}`),
			Options:       json.RawMessage(`{"out": "stderr"}`),
			MaxQueueSize:  1000,
		}
	}

	// Add file logging if logfile flag is provided
	if *logFile != "" {
		cfg["file"] = mlog.TargetCfg{
			Type:         "file",
			Levels:       mlog.StdAll,
			Format:       "json",
			Options:      json.RawMessage(fmt.Sprintf(`{"compress": false, "filename": "%s"}`, *logFile)),
			MaxQueueSize: 1000,
		}
	}

	err = logger.ConfigureTargets(cfg, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to configure logger: %v\n", err)
		os.Exit(1)
	}
	logger.RedirectStdLog(mlog.LvlStdLog)

	// Check required parameters
	if *serverURL == "" {
		// Try environment variable
		*serverURL = os.Getenv("MM_SERVER_URL")
		if *serverURL == "" {
			logger.Error("server URL is required (use -server-url or MM_SERVER_URL environment variable)")
			os.Exit(1)
		}
	}

	// Determine authentication mode
	hasTokenAuth := *token != "" || os.Getenv("MM_ACCESS_TOKEN") != ""
	hasOAuthAuth := *clientID != "" || *clientSecret != "" || *redirectURL != ""

	if !hasTokenAuth && !hasOAuthAuth {
		logger.Error("authentication is required: either PAT auth (-token) or OAuth auth (-oauth-client-id, -oauth-client-secret, -oauth-redirect-url)")
		os.Exit(1)
	}

	if hasTokenAuth && hasOAuthAuth {
		logger.Error("cannot use both PAT and OAuth authentication at the same time")
		os.Exit(1)
	}

	// Validate PAT authentication parameters
	if hasTokenAuth {
		if *token == "" {
			*token = os.Getenv("MM_ACCESS_TOKEN")
		}
		if *token == "" {
			logger.Error("personal access token is required for PAT auth (use -token or MM_ACCESS_TOKEN environment variable)")
			os.Exit(1)
		}
	}

	// Validate OAuth authentication parameters
	if hasOAuthAuth {
		if *clientID == "" || *clientSecret == "" || *redirectURL == "" {
			logger.Error("OAuth authentication requires all three parameters: -oauth-client-id, -oauth-client-secret, -oauth-redirect-url")
			os.Exit(1)
		}

		// OAuth auth requires HTTP transport
		if *transport == "stdio" {
			logger.Error("OAuth authentication requires HTTP transport (use -transport http)")
			os.Exit(1)
		}
	}

	// Validate transport type
	switch *transport {
	case "stdio":
		// stdio mode - default, no additional validation needed
	case "http":
		if *httpPort <= 0 || *httpPort > 65535 {
			logger.Error("invalid port number", mlog.Int("port", *httpPort))
			os.Exit(1)
		}
	default:
		logger.Error("unsupported transport type", mlog.String("transport", *transport))
		logger.Error("supported transport types: stdio, http")
		os.Exit(1)
	}

	// Determine authentication mode for logging
	authMode := "PAT"
	if hasOAuthAuth {
		authMode = "OAuth"
	}

	// Only log startup info in debug mode to avoid interfering with JSON-RPC for stdio
	if *debug {
		logger.Info("starting mattermost mcp server",
			mlog.String("server_url", *serverURL),
			mlog.String("transport", *transport),
			mlog.String("auth_mode", authMode),
			mlog.Bool("debug", *debug),
		)
		if *transport == "http" {
			logger.Info("http transport configuration", mlog.Int("port", *httpPort))
		}
	}

	if *devMode {
		logger.Info("development mode enabled", mlog.Bool("dev_mode", *devMode))
	}

	// Create server configuration
	config := mcpserver.Config{
		ServerURL:           *serverURL,
		PersonalAccessToken: *token,
		RequestTimeout:      defaultTimeout,
		Transport:           *transport,
		HTTPPort:            *httpPort,
		DevMode:             *devMode,
	}

	// Create authentication provider based on mode
	var authProvider mcpserver.AuthenticationProvider
	if hasTokenAuth {
		authProvider = mcpserver.NewTokenAuthenticationProvider(*serverURL, *token, logger)
	} else {
		authProvider = mcpserver.NewOAuthAuthenticationProvider(*clientID, *clientSecret, *redirectURL, *serverURL, logger)
	}

	// Create Mattermost MCP server with abstracted interface
	mcpServer := mcpserver.NewMattermostMCPServer(config, authProvider, logger)

	if *debug {
		logger.Info("starting mcp server", mlog.String("transport", *transport))
	}

	// Start the MCP server using the specified transport
	if err := mcpServer.Serve(); err != nil {
		logger.Error("server error", mlog.Err(err))
		os.Exit(1)
	}

	if *debug {
		logger.Info("mcp server stopped")
	}
}
