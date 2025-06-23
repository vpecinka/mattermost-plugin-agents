// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
)

// MCPServer defines the interface for MCP server implementations
// This abstraction allows us to swap out the underlying MCP library if needed
type MCPServer interface {
	// ServeStdio starts the server using stdio transport with proper signal handling
	ServeStdio() error

	// AddTool registers a tool with the MCP server
	AddTool(tool MCPTool, handler MCPToolHandler)

	// SetLogger configures the logger for the MCP server
	SetLogger(logger interface{})
}

// MCPTool represents a tool that can be executed by the MCP server
type MCPTool struct {
	Name        string
	Description string
	Properties  map[string]MCPProperty
	Required    []string
}

// MCPProperty represents a property in a tool's input schema
type MCPProperty struct {
	Type        string
	Description string
	Required    bool
}

// MCPToolHandler is a function that handles tool execution
type MCPToolHandler func(ctx context.Context, request MCPToolRequest) (*MCPToolResult, error)

// MCPToolRequest represents a tool execution request
type MCPToolRequest struct {
	Name      string
	Arguments map[string]interface{}
}

// MCPToolResult represents the result of a tool execution
type MCPToolResult struct {
	Content []MCPContent
	IsError bool
}

// MCPContent represents content returned by a tool
type MCPContent struct {
	Type string
	Text string
}

// Option represents configuration options for the MCP server
type Option func(MCPServer)

// WithMCPLogger sets a custom logger for the MCP server
func WithMCPLogger(logger interface{}) Option {
	return func(s MCPServer) {
		s.SetLogger(logger)
	}
}
