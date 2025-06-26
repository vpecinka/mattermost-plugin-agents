// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostMCPServer provides a high-level interface for creating an MCP server
// with Mattermost-specific tools and authentication
type MattermostMCPServer struct {
	mcpServer    *server.MCPServer
	authProvider AuthenticationProvider
	logger       mlog.LoggerIFace
	config       Config
}

// NewMattermostMCPServer creates a new Mattermost MCP server with the specified configuration
func NewMattermostMCPServer(config Config, authProvider AuthenticationProvider, logger mlog.LoggerIFace) (*MattermostMCPServer, error) {
	// Create the mcp-go server directly
	mcpServer := server.NewMCPServer(
		"mattermost-mcp-server",
		"0.1.0",
		server.WithToolCapabilities(true), // Enable tool list changed notifications
		server.WithLogging(),              // Enable logging capabilities
	)

	mattermostServer := &MattermostMCPServer{
		mcpServer:    mcpServer,
		authProvider: authProvider,
		logger:       logger,
		config:       config,
	}

	// For standalone mode (stdio with PAT), validate token at startup
	if config.Transport == "stdio" {
		if err := mattermostServer.validateTokenAtStartup(); err != nil {
			return nil, fmt.Errorf("startup token validation failed: %w", err)
		}
	}

	// Register all Mattermost tools
	mattermostServer.registerMattermostTools()

	return mattermostServer, nil
}

// validateTokenAtStartup validates the PAT token during server initialization
func (s *MattermostMCPServer) validateTokenAtStartup() error {
	// Create client with the configured token
	client := model.NewAPIv4Client(s.config.ServerURL)
	client.SetToken(s.config.PersonalAccessToken)

	s.logger.Debug("Validating token at startup", mlog.String("server_url", s.config.ServerURL))

	// Test the token with a simple GetMe call
	user, response, err := client.GetMe(context.Background(), "")
	if err != nil {
		if response != nil {
			s.logger.Error("GetMe API call failed",
				mlog.Int("status_code", response.StatusCode),
				mlog.String("server_url", s.config.ServerURL),
				mlog.Err(err))
		}
		return fmt.Errorf("token validation failed: %w", err)
	}

	s.logger.Debug("Token validation successful",
		mlog.String("user_id", user.Id),
		mlog.String("username", user.Username),
		mlog.String("email", user.Email))

	return nil
}

// Serve starts the server using the configured transport
func (s *MattermostMCPServer) Serve() error {
	switch s.config.Transport {
	case "stdio", "": // default to stdio for backward compatibility
		return s.serveStdio()
	case "http":
		return s.serveHTTP()
	default:
		return fmt.Errorf("unsupported transport type: %s", s.config.Transport)
	}
}

// serveStdio starts the server using stdio transport
func (s *MattermostMCPServer) serveStdio() error {
	// Configure error logger to use our mlog logger if available
	var errorLogger *log.Logger
	if s.logger != nil {
		// Create a custom writer that forwards to mlog
		errorLogger = log.New(&mlogWriter{logger: s.logger}, "", 0)
	} else {
		errorLogger = log.New(os.Stderr, "", log.LstdFlags)
	}

	return server.ServeStdio(s.mcpServer, server.WithErrorLogger(errorLogger))
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
	readPostTool := mcp.NewTool("read_post",
		mcp.WithDescription("Read a specific post and its thread from Mattermost"),
		mcp.WithString("post_id", mcp.Description("The ID of the post to read"), mcp.Required()),
		mcp.WithBoolean("include_thread", mcp.Description("Whether to include the entire thread (default: true)")),
	)
	s.mcpServer.AddTool(readPostTool, s.createToolHandler("read_post"))

	// Register read_channel tool
	readChannelTool := mcp.NewTool("read_channel",
		mcp.WithDescription("Read recent posts from a Mattermost channel"),
		mcp.WithString("channel_id", mcp.Description("The ID of the channel to read from"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Number of posts to retrieve (default: 20, max: 100)")),
		mcp.WithString("since", mcp.Description("Only get posts since this timestamp (ISO 8601 format)")),
	)
	s.mcpServer.AddTool(readChannelTool, s.createToolHandler("read_channel"))

	// Register search_posts tool
	searchPostsTool := mcp.NewTool("search_posts",
		mcp.WithDescription("Search for posts in Mattermost"),
		mcp.WithString("query", mcp.Description("The search query"), mcp.Required()),
		mcp.WithString("team_id", mcp.Description("Optional team ID to limit search scope")),
		mcp.WithString("channel_id", mcp.Description("Optional channel ID to limit search to a specific channel")),
		mcp.WithNumber("limit", mcp.Description("Number of results to return (default: 20, max: 100)")),
	)
	s.mcpServer.AddTool(searchPostsTool, s.createToolHandler("search_posts"))

	// Register create_post tool
	createPostTool := mcp.NewTool("create_post",
		mcp.WithDescription("Create a new post in Mattermost"),
		mcp.WithString("channel_id", mcp.Description("The ID of the channel to post in"), mcp.Required()),
		mcp.WithString("message", mcp.Description("The message content"), mcp.Required()),
		mcp.WithString("root_id", mcp.Description("Optional root post ID for replies")),
	)
	s.mcpServer.AddTool(createPostTool, s.createToolHandler("create_post"))

	// Register create_channel tool
	createChannelTool := mcp.NewTool("create_channel",
		mcp.WithDescription("Create a new channel in Mattermost"),
		mcp.WithString("name", mcp.Description("The channel name (URL-friendly)"), mcp.Required()),
		mcp.WithString("display_name", mcp.Description("The channel display name"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Channel type: 'O' for public, 'P' for private"), mcp.Required()),
		mcp.WithString("team_id", mcp.Description("The team ID where the channel will be created"), mcp.Required()),
		mcp.WithString("purpose", mcp.Description("Optional channel purpose")),
		mcp.WithString("header", mcp.Description("Optional channel header")),
	)
	s.mcpServer.AddTool(createChannelTool, s.createToolHandler("create_channel"))

	// Register get_channel_info tool
	getChannelInfoTool := mcp.NewTool("get_channel_info",
		mcp.WithDescription("Get information about a channel. If you have a channel ID, use that for fastest lookup. If the user provides a human-readable name, try channel_display_name first (what users see in the UI), then channel_name (URL name) as fallback."),
		mcp.WithString("channel_id", mcp.Description("The exact channel ID (fastest, most reliable method)")),
		mcp.WithString("channel_display_name", mcp.Description("The human-readable display name users see (e.g. 'General Discussion')")),
		mcp.WithString("channel_name", mcp.Description("The URL-friendly channel name (e.g. 'general-discussion')")),
		mcp.WithString("team_id", mcp.Description("Team ID (required if using channel_name or channel_display_name)")),
	)
	s.mcpServer.AddTool(getChannelInfoTool, s.createToolHandler("get_channel_info"))

	// Register get_team_info tool
	getTeamInfoTool := mcp.NewTool("get_team_info",
		mcp.WithDescription("Get information about a team. If you have a team ID, use that for fastest lookup. If the user provides a human-readable name, try team_display_name first (what users see in the UI), then team_name (URL name) as fallback."),
		mcp.WithString("team_id", mcp.Description("The exact team ID (fastest, most reliable method)")),
		mcp.WithString("team_display_name", mcp.Description("The human-readable display name users see (e.g. 'Engineering Team')")),
		mcp.WithString("team_name", mcp.Description("The URL-friendly team name (e.g. 'engineering-team')")),
	)
	s.mcpServer.AddTool(getTeamInfoTool, s.createToolHandler("get_team_info"))

	// Register search_users tool
	searchUsersTool := mcp.NewTool("search_users",
		mcp.WithDescription("Search for existing users by username, email, or name"),
		mcp.WithString("term", mcp.Description("Search term (username, email, first name, or last name)"), mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("Maximum number of results to return (default: 20, max: 100)")),
	)
	s.mcpServer.AddTool(searchUsersTool, s.createToolHandler("search_users"))

	// Register get_channel_members tool
	getChannelMembersTool := mcp.NewTool("get_channel_members",
		mcp.WithDescription("Get all members of a channel"),
		mcp.WithString("channel_id", mcp.Description("ID of the channel to get members for"), mcp.Required()),
	)
	s.mcpServer.AddTool(getChannelMembersTool, s.createToolHandler("get_channel_members"))

	// Register get_team_members tool
	getTeamMembersTool := mcp.NewTool("get_team_members",
		mcp.WithDescription("Get all members of a team"),
		mcp.WithString("team_id", mcp.Description("ID of the team to get members for"), mcp.Required()),
	)
	s.mcpServer.AddTool(getTeamMembersTool, s.createToolHandler("get_team_members"))

	// Register development tools if dev mode is enabled
	if s.config.DevMode {
		s.registerDevTools()
	}
}

// registerDevTools registers development-specific tools when dev mode is enabled
func (s *MattermostMCPServer) registerDevTools() {
	// Register create_user tool
	createUserTool := mcp.NewTool("create_user",
		mcp.WithDescription("Create a new user account (dev mode only)"),
		mcp.WithString("username", mcp.Description("Username for the new user"), mcp.Required()),
		mcp.WithString("email", mcp.Description("Email address for the new user"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password for the new user"), mcp.Required()),
		mcp.WithString("first_name", mcp.Description("First name of the user")),
		mcp.WithString("last_name", mcp.Description("Last name of the user")),
		mcp.WithString("nickname", mcp.Description("Nickname for the user")),
	)
	s.mcpServer.AddTool(createUserTool, s.createToolHandler("create_user"))

	// Register create_team tool
	createTeamTool := mcp.NewTool("create_team",
		mcp.WithDescription("Create a new team (dev mode only)"),
		mcp.WithString("name", mcp.Description("URL name for the team"), mcp.Required()),
		mcp.WithString("display_name", mcp.Description("Display name for the team"), mcp.Required()),
		mcp.WithString("type", mcp.Description("Team type: 'O' for open, 'I' for invite only"), mcp.Required()),
		mcp.WithString("description", mcp.Description("Team description")),
	)
	s.mcpServer.AddTool(createTeamTool, s.createToolHandler("create_team"))

	// Register add_user_to_team tool
	addUserToTeamTool := mcp.NewTool("add_user_to_team",
		mcp.WithDescription("Add a user to a team (dev mode only)"),
		mcp.WithString("user_id", mcp.Description("ID of the user to add"), mcp.Required()),
		mcp.WithString("team_id", mcp.Description("ID of the team to add user to"), mcp.Required()),
	)
	s.mcpServer.AddTool(addUserToTeamTool, s.createToolHandler("add_user_to_team"))

	// Register add_user_to_channel tool
	addUserToChannelTool := mcp.NewTool("add_user_to_channel",
		mcp.WithDescription("Add a user to a channel (dev mode only)"),
		mcp.WithString("user_id", mcp.Description("ID of the user to add"), mcp.Required()),
		mcp.WithString("channel_id", mcp.Description("ID of the channel to add user to"), mcp.Required()),
	)
	s.mcpServer.AddTool(addUserToChannelTool, s.createToolHandler("add_user_to_channel"))

	// Register create_post_as_user tool
	createPostAsUserTool := mcp.NewTool("create_post_as_user",
		mcp.WithDescription("Create a post as a specific user using username/password login. Use this tool in dev mode for creating realistic multi-user scenarios. Simply provide the username and password of created users."),
		mcp.WithString("username", mcp.Description("Username to login as"), mcp.Required()),
		mcp.WithString("password", mcp.Description("Password to login with"), mcp.Required()),
		mcp.WithString("channel_id", mcp.Description("The ID of the channel to post in"), mcp.Required()),
		mcp.WithString("message", mcp.Description("The message content"), mcp.Required()),
		mcp.WithString("root_id", mcp.Description("Optional root post ID for replies")),
		mcp.WithString("props", mcp.Description("Optional post properties (JSON string)")),
	)
	s.mcpServer.AddTool(createPostAsUserTool, s.createToolHandler("create_post_as_user"))
}

func (s *MattermostMCPServer) createToolHandler(toolName string) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get authenticated client
		client, _, err := s.getAuthenticatedClient(ctx)
		if err != nil {
			s.logger.Debug("Tool call failed",
				mlog.String("tool", toolName),
				mlog.Err(err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: " + err.Error(),
					},
				},
				IsError: true,
			}, nil
		}

		// No need for user context - Mattermost gets userID from session
		ctxWithUser := ctx

		// Use our existing tool provider to execute the tool
		toolProvider := NewMattermostToolProvider(s.authProvider, s.logger)

		// Create dev tool provider for development tools
		devToolProvider := NewDevToolProvider(s.authProvider, s.logger, s.config.ServerURL)

		// Execute the tool using our existing implementation
		var result *mcp.CallToolResult
		switch toolName {
		case "read_post":
			result, err = toolProvider.readPost(ctxWithUser, client, request.Params.Arguments)
		case "read_channel":
			result, err = toolProvider.readChannel(ctxWithUser, client, request.Params.Arguments)
		case "search_posts":
			result, err = toolProvider.searchPosts(ctxWithUser, client, request.Params.Arguments)
		case "create_post":
			result, err = toolProvider.createPost(ctxWithUser, client, request.Params.Arguments)
		case "create_channel":
			result, err = toolProvider.createChannel(ctxWithUser, client, request.Params.Arguments)
		case "get_channel_info":
			result, err = toolProvider.getChannelInfo(ctxWithUser, client, request.Params.Arguments)
		case "get_team_info":
			result, err = toolProvider.getTeamInfo(ctxWithUser, client, request.Params.Arguments)
		case "search_users":
			result, err = toolProvider.searchUsers(ctxWithUser, client, request.Params.Arguments)
		case "get_channel_members":
			result, err = toolProvider.getChannelMembers(ctxWithUser, client, request.Params.Arguments)
		case "get_team_members":
			result, err = toolProvider.getTeamMembers(ctxWithUser, client, request.Params.Arguments)
		// Development tools (only available in dev mode)
		case "create_user":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: create_user tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.createUser(ctxWithUser, client, request.Params.Arguments)
		case "create_team":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: create_team tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.createTeam(ctxWithUser, client, request.Params.Arguments)
		case "add_user_to_team":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: add_user_to_team tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.addUserToTeam(ctxWithUser, client, request.Params.Arguments)
		case "add_user_to_channel":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: add_user_to_channel tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.addUserToChannel(ctxWithUser, client, request.Params.Arguments)
		case "create_post_as_user":
			if !s.config.DevMode {
				s.logger.Debug("Tool call failed - dev mode required",
					mlog.String("tool", toolName))
				return &mcp.CallToolResult{
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Error: create_post_as_user tool is only available in development mode",
						},
					},
					IsError: true,
				}, nil
			}
			result, err = devToolProvider.createPostAsUser(ctxWithUser, request.Params.Arguments)
		default:
			s.logger.Debug("Tool call failed - unknown tool",
				mlog.String("tool", toolName))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: unknown tool: " + toolName,
					},
				},
				IsError: true,
			}, nil
		}

		if err != nil {
			s.logger.Debug("Tool call failed - execution error",
				mlog.String("tool", toolName),
				mlog.Err(err))
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: " + err.Error(),
					},
				},
				IsError: true,
			}, nil
		}

		// Log successful tool completion
		isError := result != nil && result.IsError
		if isError {
			s.logger.Debug("Tool call completed with error result",
				mlog.String("tool", toolName))
		} else {
			s.logger.Debug("Tool call completed successfully",
				mlog.String("tool", toolName))
		}

		// Result is already a *mcp.CallToolResult
		return result, nil
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

	// Create client directly - no validation needed since Mattermost APIs will validate
	client := model.NewAPIv4Client(s.config.ServerURL)
	client.SetToken(token)

	return client, "", nil // userID not needed - Mattermost gets it from session
}

// mlogWriter adapts mlog.LoggerIFace to io.Writer for the mcp-go error logger
type mlogWriter struct {
	logger mlog.LoggerIFace
}

func (w *mlogWriter) Write(p []byte) (n int, err error) {
	if w.logger != nil {
		w.logger.Error(string(p))
	}
	return len(p), nil
}

// CallToolForTest calls a tool handler directly for testing purposes
// This bypasses the MCP transport layer and calls the tool implementation directly
func (s *MattermostMCPServer) CallToolForTest(ctx context.Context, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	// Create a proper MCP CallToolRequest
	request := mcp.CallToolRequest{}
	request.Params.Name = toolName
	request.Params.Arguments = arguments

	// Get and call the tool handler
	toolHandler := s.createToolHandler(toolName)
	return toolHandler(ctx, request)
}
