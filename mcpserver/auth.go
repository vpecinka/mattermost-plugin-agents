// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// TokenAuthenticationProvider provides authentication using tokens (PAT or OAuth)
type TokenAuthenticationProvider struct {
	serverURL    string
	defaultToken string // For stdio mode
	logger       mlog.LoggerIFace
}

// NewTokenAuthenticationProvider creates a new token authentication provider
func NewTokenAuthenticationProvider(serverURL, defaultToken string, logger mlog.LoggerIFace) *TokenAuthenticationProvider {
	return &TokenAuthenticationProvider{
		serverURL:    serverURL,
		defaultToken: defaultToken,
		logger:       logger,
	}
}

// ValidateAuth validates a token and returns the associated user ID
func (p *TokenAuthenticationProvider) ValidateAuth(ctx context.Context, token string) (string, error) {
	// Use default token if none provided (for stdio mode)
	if token == "" {
		token = p.defaultToken
	}

	if token == "" {
		return "", fmt.Errorf("no authentication token provided")
	}

	// Create client and validate token once
	client := model.NewAPIv4Client(p.serverURL)
	client.SetToken(token)

	// Get current user to validate token
	user, _, err := client.GetMe(ctx, "")
	if err != nil {
		p.logger.Error("failed to validate token", mlog.Err(err))
		return "", fmt.Errorf("invalid authentication token: %w", err)
	}

	p.logger.Debug("validated token for user", mlog.String("user_id", user.Id), mlog.String("username", user.Username))

	return user.Id, nil
}

// GetMattermostClient returns an authenticated Mattermost client for a user
func (p *TokenAuthenticationProvider) GetMattermostClient(ctx context.Context, userID string, token string) (*model.Client4, error) {
	// Use default token if none provided (for stdio mode)
	if token == "" {
		token = p.defaultToken
	}

	if token == "" {
		return nil, fmt.Errorf("no authentication token provided")
	}

	// Create a fresh client for each request - no caching
	client := model.NewAPIv4Client(p.serverURL)
	client.SetToken(token)

	// For standalone mode, return the client4 directly - no wrapper needed!
	return client, nil
}

// OAuthAuthenticationProvider will provide OAuth authentication for HTTP transport
// TODO: Implement when HTTP transport is added
type OAuthAuthenticationProvider struct {
	clientID     string
	clientSecret string
	redirectURL  string
	serverURL    string
	logger       mlog.LoggerIFace
}

// NewOAuthAuthenticationProvider creates a new OAuth authentication provider
// TODO: Implement when HTTP transport is added
func NewOAuthAuthenticationProvider(clientID, clientSecret, redirectURL, serverURL string, logger mlog.LoggerIFace) *OAuthAuthenticationProvider {
	return &OAuthAuthenticationProvider{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		serverURL:    serverURL,
		logger:       logger,
	}
}

// ValidateAuth validates OAuth token and returns user ID
// TODO: Implement when HTTP transport is added
func (p *OAuthAuthenticationProvider) ValidateAuth(ctx context.Context, token string) (string, error) {
	return "", fmt.Errorf("OAuth authentication not yet implemented")
}

// GetMattermostClient returns an OAuth-authenticated Mattermost client
// TODO: Implement when HTTP transport is added
func (p *OAuthAuthenticationProvider) GetMattermostClient(ctx context.Context, userID string, token string) (*model.Client4, error) {
	return nil, fmt.Errorf("OAuth authentication not yet implemented")
}
