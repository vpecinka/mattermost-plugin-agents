// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	mmcontainer "github.com/mattermost/testcontainers-mattermost-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-ai/mcpserver"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// TestSuite represents the integration test suite
type TestSuite struct {
	t          *testing.T
	container  *mmcontainer.MattermostContainer
	serverURL  string
	adminToken string
	logger     mlog.LoggerIFace
	mcpServer  *mcpserver.MattermostMCPServer
	devMode    bool
}

// SetupTestSuite initializes a Mattermost container and MCP server for testing
func SetupTestSuite(t *testing.T) *TestSuite {
	ctx := context.Background()

	// Start Mattermost container with PAT enabled
	container, err := mmcontainer.RunContainer(ctx,
		mmcontainer.WithLicense(""),
	)
	require.NoError(t, err, "Failed to start Mattermost container")

	// Enable personal access tokens in the server config
	err = container.SetConfig(ctx, "ServiceSettings.EnableUserAccessTokens", "true")
	require.NoError(t, err, "Failed to enable personal access tokens")

	// Get connection details
	serverURL, err := container.URL(ctx)
	require.NoError(t, err, "Failed to get server URL")

	// Get admin client and create a PAT token
	adminClient, err := container.GetAdminClient(ctx)
	require.NoError(t, err, "Failed to get admin client")

	// Create a personal access token for testing
	pat, _, err := adminClient.CreateUserAccessToken(ctx, "me", "MCP Integration Test Token")
	require.NoError(t, err, "Failed to create PAT token")
	adminToken := pat.Token

	// Set up logger for testing
	logger, err := mlog.NewLogger()
	require.NoError(t, err, "Failed to create logger")

	cfg := make(mlog.LoggerConfiguration)
	cfg["console"] = mlog.TargetCfg{
		Type:          "console",
		Levels:        []mlog.Level{mlog.LvlDebug, mlog.LvlInfo, mlog.LvlWarn, mlog.LvlError},
		Format:        "plain",
		FormatOptions: json.RawMessage(`{"enable_color": false}`),
		Options:       json.RawMessage(`{"out": "stderr"}`),
		MaxQueueSize:  1000,
	}
	err = logger.ConfigureTargets(cfg, nil)
	require.NoError(t, err, "Failed to configure logger")

	return &TestSuite{
		t:          t,
		container:  container,
		serverURL:  serverURL,
		adminToken: adminToken,
		logger:     logger,
	}
}

// TearDown cleans up the test suite
func (suite *TestSuite) TearDown() {
	if suite.container != nil {
		ctx := context.Background()
		if err := suite.container.Terminate(ctx); err != nil {
			suite.t.Logf("Failed to terminate container: %v", err)
		}
	}
	if suite.logger != nil {
		suite.logger.Flush()
	}
}

// CreateMCPServer creates and configures an MCP server for testing
func (suite *TestSuite) CreateMCPServer(devMode bool) {
	require.NotNil(suite.t, suite.logger, "Logger must be initialized")
	require.NotEmpty(suite.t, suite.serverURL, "Server URL must be set")
	require.NotEmpty(suite.t, suite.adminToken, "Admin token must be set")

	config := mcpserver.Config{
		ServerURL:           suite.serverURL,
		PersonalAccessToken: suite.adminToken,
		RequestTimeout:      30 * time.Second,
		Transport:           "stdio",
		DevMode:             devMode,
	}

	authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)

	mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)
	require.NoError(suite.t, err, "Failed to create MCP server")

	suite.mcpServer = mcpServer
	suite.devMode = devMode
}

// TestMCPServerCreation tests basic MCP server creation and startup
func TestMCPServerCreation(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("CreateMCPServer", func(t *testing.T) {
		suite.CreateMCPServer(false)
		assert.NotNil(t, suite.mcpServer, "MCP server should be created")
	})

	t.Run("CreateMCPServerWithDevMode", func(t *testing.T) {
		suite.CreateMCPServer(true)
		assert.NotNil(t, suite.mcpServer, "MCP server with dev mode should be created")
	})
}

// TestMCPServerConfiguration tests various configuration scenarios
func TestMCPServerConfiguration(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("ValidConfiguration", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)
		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		require.NoError(t, err, "Valid configuration should not return error")
		assert.NotNil(t, mcpServer, "MCP server should be created with valid config")
	})

	t.Run("InvalidServerURL", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           "http://invalid-server-url:9999",
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider("http://invalid-server-url:9999", suite.adminToken, suite.logger)
		_, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		assert.Error(t, err, "Invalid server URL should return error")
		assert.Contains(t, err.Error(), "startup token validation failed", "Error should mention token validation failure")
	})

	t.Run("InvalidToken", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: "invalid-token-12345",
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, "invalid-token-12345", suite.logger)
		_, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		assert.Error(t, err, "Invalid token should return error")
		assert.Contains(t, err.Error(), "startup token validation failed", "Error should mention token validation failure")
	})

	t.Run("EmptyToken", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: "",
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, "", suite.logger)
		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		// Empty token should not fail at creation time (validation is skipped)
		require.NoError(t, err, "Empty token should not fail at creation")
		assert.NotNil(t, mcpServer, "MCP server should be created even with empty token")
	})

	t.Run("DevModeConfiguration", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             true,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)
		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		require.NoError(t, err, "Dev mode configuration should not return error")
		assert.NotNil(t, mcpServer, "MCP server should be created with dev mode")
	})

	t.Run("HTTPTransportConfiguration", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      30 * time.Second,
			Transport:           "http",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)
		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		require.NoError(t, err, "HTTP transport configuration should not fail at creation")
		assert.NotNil(t, mcpServer, "MCP server should be created with HTTP transport config")

		// Note: HTTP transport serving will fail since it's not implemented yet
		// but configuration creation should succeed
	})
}

// TestAuthentication tests authentication scenarios
func TestAuthentication(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("TokenAuthenticationProvider", func(t *testing.T) {
		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)
		assert.NotNil(t, authProvider, "Token authentication provider should be created")

		// Test token validation
		ctx := context.Background()
		userID, err := authProvider.ValidateAuth(ctx, suite.adminToken)
		require.NoError(t, err, "Should validate authentication")
		assert.NotEmpty(t, userID, "User ID should not be empty")
	})

	t.Run("TokenValidationAtStartup", func(t *testing.T) {
		// This tests the startup token validation that happens in NewMattermostMCPServer
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)
		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		require.NoError(t, err, "Startup token validation should succeed with valid token")
		assert.NotNil(t, mcpServer, "MCP server should be created after successful token validation")
	})

	t.Run("TokenAuthenticationFailure", func(t *testing.T) {
		invalidToken := "invalid-token-xyz"
		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, invalidToken, suite.logger)

		ctx := context.Background()
		_, err := authProvider.ValidateAuth(ctx, invalidToken)
		assert.Error(t, err, "Invalid token should fail validation")
	})
}

// TestMCPServerStartupValidation tests server startup validation scenarios
func TestMCPServerStartupValidation(t *testing.T) {
	suite := SetupTestSuite(t)
	defer suite.TearDown()

	t.Run("SuccessfulStartupValidation", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, suite.adminToken, suite.logger)

		// This internally calls validateTokenAtStartup
		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		require.NoError(t, err, "Startup validation should succeed")
		assert.NotNil(t, mcpServer, "MCP server should be created after successful validation")
	})

	t.Run("StartupValidationWithInvalidServer", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           "http://nonexistent-server:8065",
			PersonalAccessToken: suite.adminToken,
			RequestTimeout:      5 * time.Second, // Shorter timeout for faster test
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider("http://nonexistent-server:8065", suite.adminToken, suite.logger)

		_, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		assert.Error(t, err, "Startup validation should fail with invalid server")
		assert.Contains(t, err.Error(), "startup token validation failed", "Error should mention startup validation failure")
	})

	t.Run("StartupValidationWithUnauthorizedToken", func(t *testing.T) {
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: "unauthorized-token-123",
			RequestTimeout:      30 * time.Second,
			Transport:           "stdio",
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, "unauthorized-token-123", suite.logger)

		_, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		assert.Error(t, err, "Startup validation should fail with unauthorized token")
		assert.Contains(t, err.Error(), "startup token validation failed", "Error should mention startup validation failure")
	})

	t.Run("NoStartupValidationForEmptyToken", func(t *testing.T) {
		// When transport is not stdio or token is empty, startup validation is skipped
		config := mcpserver.Config{
			ServerURL:           suite.serverURL,
			PersonalAccessToken: "",
			RequestTimeout:      30 * time.Second,
			Transport:           "http", // Non-stdio transport
			DevMode:             false,
		}

		authProvider := mcpserver.NewTokenAuthenticationProvider(suite.serverURL, "", suite.logger)

		mcpServer, err := mcpserver.NewMattermostMCPServer(config, authProvider, suite.logger)

		require.NoError(t, err, "Server creation should succeed when startup validation is skipped")
		assert.NotNil(t, mcpServer, "MCP server should be created")
	})
}
