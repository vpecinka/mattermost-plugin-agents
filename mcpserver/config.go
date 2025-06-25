// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// Context key types
type contextKey string

const (
	UserIDKey contextKey = "userID"
	TokenKey  contextKey = "token"
)

// Config represents the configuration for the MCP server
type Config struct {
	// Mattermost server URL (e.g., "https://mattermost.company.com")
	ServerURL string `json:"server_url"`

	// Personal Access Token for authentication
	PersonalAccessToken string `json:"personal_access_token"`

	// Optional headers to include in requests
	Headers map[string]string `json:"headers,omitempty"`

	// Timeout for requests to Mattermost
	RequestTimeout time.Duration `json:"request_timeout"`

	// Transport type (stdio, http)
	Transport string `json:"transport"`

	// HTTP port for http transport
	HTTPPort int `json:"http_port"`

	// Development mode enables additional tools for setting up test data
	DevMode bool `json:"dev_mode"`
}

// AuthenticationProvider handles authentication for MCP requests
type AuthenticationProvider interface {
	// ValidateAuth validates the authentication and returns user ID
	ValidateAuth(ctx context.Context, token string) (string, error)

	// GetMattermostClient returns an authenticated Mattermost client for a user
	GetMattermostClient(ctx context.Context, userID string, token string) (*model.Client4, error)
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

// Content represents content returned by a tool
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	Data string `json:"data,omitempty"`
}
