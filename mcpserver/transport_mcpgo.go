// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// mcpGoServer implements MCPServer using the mcp-go library
type mcpGoServer struct {
	server *server.MCPServer
	logger mlog.LoggerIFace
}

// NewMCPGoServer creates a new MCP server using the mcp-go library
func NewMCPGoServer(name, version string, opts ...Option) MCPServer {
	mcpServer := server.NewMCPServer(
		name,
		version,
		server.WithToolCapabilities(true), // Enable tool list changed notifications
		server.WithLogging(),              // Enable logging capabilities
	)

	impl := &mcpGoServer{
		server: mcpServer,
	}

	// Apply options
	for _, opt := range opts {
		opt(impl)
	}

	return impl
}

// ServeStdio starts the server using stdio transport
func (s *mcpGoServer) ServeStdio() error {
	// Configure error logger to use our mlog logger if available
	var errorLogger *log.Logger
	if s.logger != nil {
		// Create a custom writer that forwards to mlog
		errorLogger = log.New(&mlogWriter{logger: s.logger}, "", 0)
	} else {
		errorLogger = log.New(os.Stderr, "", log.LstdFlags)
	}

	return server.ServeStdio(s.server, server.WithErrorLogger(errorLogger))
}

// AddTool registers a tool with the MCP server
func (s *mcpGoServer) AddTool(tool MCPTool, handler MCPToolHandler) {
	// Convert our MCPTool to mcp-go Tool
	var options []mcp.ToolOption

	if tool.Description != "" {
		options = append(options, mcp.WithDescription(tool.Description))
	}

	// Add properties
	for name, prop := range tool.Properties {
		var propOpts []mcp.PropertyOption
		if prop.Description != "" {
			propOpts = append(propOpts, mcp.Description(prop.Description))
		}
		if prop.Required {
			propOpts = append(propOpts, mcp.Required())
		}

		switch prop.Type {
		case "string":
			options = append(options, mcp.WithString(name, propOpts...))
		case "number", "integer":
			options = append(options, mcp.WithNumber(name, propOpts...))
		case "boolean":
			options = append(options, mcp.WithBoolean(name, propOpts...))
		default:
			// Default to string for unknown types
			options = append(options, mcp.WithString(name, propOpts...))
		}
	}

	mcpTool := mcp.NewTool(tool.Name, options...)

	// Create handler wrapper
	mcpHandler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Convert mcp-go request to our interface
		mcpRequest := MCPToolRequest{
			Name:      request.Params.Name,
			Arguments: request.Params.Arguments,
		}

		// Call our handler
		result, err := handler(ctx, mcpRequest)
		if err != nil {
			// Return error as content for better LLM visibility
			errorResult := &mcp.CallToolResult{
				Content: []mcp.Content{
					mcp.TextContent{
						Type: "text",
						Text: "Error: " + err.Error(),
					},
				},
				IsError: true,
			}
			return errorResult, nil
		}

		// Convert our result to mcp-go format
		var content []mcp.Content
		for _, c := range result.Content {
			content = append(content, mcp.TextContent{
				Type: "text",
				Text: c.Text,
			})
		}

		mcpResult := &mcp.CallToolResult{
			Content: content,
			IsError: result.IsError,
		}

		return mcpResult, nil
	}

	s.server.AddTool(mcpTool, mcpHandler)
}

// SetLogger configures the logger for the MCP server
func (s *mcpGoServer) SetLogger(logger interface{}) {
	if mlogLogger, ok := logger.(mlog.LoggerIFace); ok {
		s.logger = mlogLogger
	}
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
