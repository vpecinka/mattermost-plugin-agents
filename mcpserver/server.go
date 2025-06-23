// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostMCPServer provides a high-level interface for creating an MCP server
// with Mattermost-specific tools and authentication
type MattermostMCPServer struct {
	mcpServer    MCPServer
	authProvider AuthenticationProvider
	logger       mlog.LoggerIFace
	config       Config
}

// NewMattermostMCPServer creates a new Mattermost MCP server with the specified configuration
func NewMattermostMCPServer(config Config, authProvider AuthenticationProvider, logger mlog.LoggerIFace) *MattermostMCPServer {
	// Create the underlying MCP server using our interface
	mcpServer := NewMCPGoServer(
		"mattermost-mcp-server",
		"0.1.0",
		WithMCPLogger(logger),
	)

	server := &MattermostMCPServer{
		mcpServer:    mcpServer,
		authProvider: authProvider,
		logger:       logger,
		config:       config,
	}

	// Register all Mattermost tools
	server.registerMattermostTools()

	return server
}

// Serve starts the server using the configured transport
func (s *MattermostMCPServer) Serve() error {
	switch s.config.Transport {
	case "stdio", "": // default to stdio for backward compatibility
		return s.mcpServer.ServeStdio()
	case "http":
		return s.serveHTTP()
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport)
	}
}

// ServeStdio starts the server using stdio transport (kept for backward compatibility)
func (s *MattermostMCPServer) ServeStdio() error {
	return s.mcpServer.ServeStdio()
}

// serveHTTP starts the server using HTTP transport
func (s *MattermostMCPServer) serveHTTP() error {
	// TODO: Implement HTTP/SSE transport for OAuth authentication
	// This will be implemented when OAuth support is added
	s.logger.Info("HTTP transport requested but not yet implemented")
	s.logger.Info("Future implementation will support OAuth authentication and StreamableHTTP")
	return fmt.Errorf("HTTP transport not yet implemented - will be added for OAuth support")
}

// registerMattermostTools registers all Mattermost tools with the MCP server
func (s *MattermostMCPServer) registerMattermostTools() {
	// Register read_post tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "read_post",
			Description: "Read a specific post and its thread from Mattermost",
			Properties: map[string]MCPProperty{
				"post_id": {
					Type:        "string",
					Description: "The ID of the post to read",
					Required:    true,
				},
				"include_thread": {
					Type:        "boolean",
					Description: "Whether to include the entire thread (default: true)",
				},
			},
			Required: []string{"post_id"},
		},
		s.createToolHandler("read_post"),
	)

	// Register read_channel tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "read_channel",
			Description: "Read recent posts from a Mattermost channel",
			Properties: map[string]MCPProperty{
				"channel_id": {
					Type:        "string",
					Description: "The ID of the channel to read from",
					Required:    true,
				},
				"limit": {
					Type:        "number",
					Description: "Number of posts to retrieve (default: 20, max: 100)",
				},
				"since": {
					Type:        "string",
					Description: "Only get posts since this timestamp (ISO 8601 format)",
				},
			},
			Required: []string{"channel_id"},
		},
		s.createToolHandler("read_channel"),
	)

	// Register search_posts tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "search_posts",
			Description: "Search for posts in Mattermost",
			Properties: map[string]MCPProperty{
				"query": {
					Type:        "string",
					Description: "The search query",
					Required:    true,
				},
				"team_id": {
					Type:        "string",
					Description: "Optional team ID to limit search scope",
				},
				"channel_id": {
					Type:        "string",
					Description: "Optional channel ID to limit search to a specific channel",
				},
				"limit": {
					Type:        "number",
					Description: "Number of results to return (default: 20, max: 100)",
				},
			},
			Required: []string{"query"},
		},
		s.createToolHandler("search_posts"),
	)

	// Register create_post tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "create_post",
			Description: "Create a new post in Mattermost",
			Properties: map[string]MCPProperty{
				"channel_id": {
					Type:        "string",
					Description: "The ID of the channel to post in",
					Required:    true,
				},
				"message": {
					Type:        "string",
					Description: "The message content",
					Required:    true,
				},
				"root_id": {
					Type:        "string",
					Description: "Optional root post ID for replies",
				},
			},
			Required: []string{"channel_id", "message"},
		},
		s.createToolHandler("create_post"),
	)

	// Register create_channel tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "create_channel",
			Description: "Create a new channel in Mattermost",
			Properties: map[string]MCPProperty{
				"name": {
					Type:        "string",
					Description: "The channel name (URL-friendly)",
					Required:    true,
				},
				"display_name": {
					Type:        "string",
					Description: "The channel display name",
					Required:    true,
				},
				"type": {
					Type:        "string",
					Description: "Channel type: 'O' for public, 'P' for private",
					Required:    true,
				},
				"team_id": {
					Type:        "string",
					Description: "The team ID where the channel will be created",
					Required:    true,
				},
				"purpose": {
					Type:        "string",
					Description: "Optional channel purpose",
				},
				"header": {
					Type:        "string",
					Description: "Optional channel header",
				},
			},
			Required: []string{"name", "display_name", "type", "team_id"},
		},
		s.createToolHandler("create_channel"),
	)

	// Register get_channel_info tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "get_channel_info",
			Description: "Get information about a channel",
			Properties: map[string]MCPProperty{
				"channel_id": {
					Type:        "string",
					Description: "The ID of the channel",
				},
				"channel_name": {
					Type:        "string",
					Description: "The name of the channel (if ID not provided)",
				},
				"team_id": {
					Type:        "string",
					Description: "Team ID (required if using channel_name)",
				},
			},
		},
		s.createToolHandler("get_channel_info"),
	)

	// Register get_team_info tool
	s.mcpServer.AddTool(
		MCPTool{
			Name:        "get_team_info",
			Description: "Get information about a team by name or display name",
			Properties: map[string]MCPProperty{
				"team_id": {
					Type:        "string",
					Description: "The ID of the team",
				},
				"team_name": {
					Type:        "string",
					Description: "The name (URL name) of the team (if ID not provided)",
				},
				"team_display_name": {
					Type:        "string",
					Description: "The display name of the team (if ID and name not provided)",
				},
			},
		},
		s.createToolHandler("get_team_info"),
	)
}

// createToolHandler creates a tool handler that bridges to our existing tool implementation
func (s *MattermostMCPServer) createToolHandler(toolName string) MCPToolHandler {
	return func(ctx context.Context, request MCPToolRequest) (*MCPToolResult, error) {
		// Get authenticated client
		client, userID, err := s.getAuthenticatedClient(ctx)
		if err != nil {
			return &MCPToolResult{
				Content: []MCPContent{{
					Type: "text",
					Text: "Error: " + err.Error(),
				}},
				IsError: true,
			}, nil
		}

		// Add user context
		ctxWithUser := context.WithValue(ctx, UserIDKey, userID)

		// Use our existing tool provider to execute the tool
		toolProvider := NewMattermostToolProvider(s.authProvider, s.logger)

		// Execute the tool using our existing implementation
		var result *ToolResult
		switch toolName {
		case "read_post":
			result, err = toolProvider.readPost(ctxWithUser, client, request.Arguments)
		case "read_channel":
			result, err = toolProvider.readChannel(ctxWithUser, client, request.Arguments)
		case "search_posts":
			result, err = toolProvider.searchPosts(ctxWithUser, client, request.Arguments)
		case "create_post":
			result, err = toolProvider.createPost(ctxWithUser, client, request.Arguments)
		case "create_channel":
			result, err = toolProvider.createChannel(ctxWithUser, client, request.Arguments)
		case "get_channel_info":
			result, err = toolProvider.getChannelInfo(ctxWithUser, client, request.Arguments)
		case "get_team_info":
			result, err = toolProvider.getTeamInfo(ctxWithUser, client, request.Arguments)
		default:
			return &MCPToolResult{
				Content: []MCPContent{{
					Type: "text",
					Text: "Error: unknown tool: " + toolName,
				}},
				IsError: true,
			}, nil
		}

		if err != nil {
			return &MCPToolResult{
				Content: []MCPContent{{
					Type: "text",
					Text: "Error: " + err.Error(),
				}},
				IsError: true,
			}, nil
		}

		// Convert legacy ToolResult to MCPToolResult
		var content []MCPContent
		for _, c := range result.Content {
			content = append(content, MCPContent{
				Type: "text",
				Text: c.Text,
			})
		}

		return &MCPToolResult{
			Content: content,
			IsError: result.IsError,
		}, nil
	}
}

// getAuthenticatedClient gets an authenticated client for the request
func (s *MattermostMCPServer) getAuthenticatedClient(ctx context.Context) (*model.Client4, string, error) {
	// For OAuth mode, token must come from request context (set by HTTP transport)
	// For PAT mode, token can come from context or fall back to config
	var token string
	if ctxToken, ok := ctx.Value(TokenKey).(string); ok && ctxToken != "" {
		token = ctxToken
	} else if s.config.PersonalAccessToken != "" {
		// Fall back to config token for PAT mode
		token = s.config.PersonalAccessToken
	}

	if token == "" {
		return nil, "", fmt.Errorf("no authentication token available - ensure token is provided via context for OAuth or config for PAT")
	}

	// Validate token and get user ID
	userID, err := s.authProvider.ValidateAuth(ctx, token)
	if err != nil {
		return nil, "", fmt.Errorf("authentication failed: %w", err)
	}

	// Get authenticated client
	client, err := s.authProvider.GetMattermostClient(ctx, userID, token)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get authenticated client: %w", err)
	}

	return client, userID, nil
}
