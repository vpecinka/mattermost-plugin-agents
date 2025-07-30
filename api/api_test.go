// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/conversations"
	"github.com/mattermost/mattermost-plugin-ai/embeddings/mocks"
	"github.com/mattermost/mattermost-plugin-ai/enterprise"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/mcp"
	"github.com/mattermost/mattermost-plugin-ai/metrics"
	"github.com/mattermost/mattermost-plugin-ai/search"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type TestEnvironment struct {
	api     *API
	mockAPI *plugintest.API
	bots    *bots.MMBots
}

// testConfigImpl is a minimal implementation of Config for testing
type testConfigImpl struct{}

func (tc *testConfigImpl) GetDefaultBotName() string {
	return "ai"
}

func (tc *testConfigImpl) MCP() mcp.Config {
	return mcp.Config{}
}

// mockMCPClientManager is a minimal implementation of MCPClientManager for testing
type mockMCPClientManager struct{}

func (m *mockMCPClientManager) GetOAuthManager() *mcp.OAuthManager {
	return nil
}

func (m *mockMCPClientManager) ProcessOAuthCallback(ctx context.Context, loggedInUserID, state, code string) (*mcp.OAuthSession, error) {
	return nil, nil
}

func (e *TestEnvironment) Cleanup(t *testing.T) {
	if e.mockAPI != nil {
		e.mockAPI.AssertExpectations(t)
	}
}

// createTestBots creates a test MMBots instance for testing
func createTestBots(mockAPI *plugintest.API, client *pluginapi.Client) *bots.MMBots {
	licenseChecker := enterprise.NewLicenseChecker(client)
	testBots := bots.New(mockAPI, client, licenseChecker, nil, &http.Client{})
	return testBots
}

// setupTestBot configures a test bot in the environment
func (e *TestEnvironment) setupTestBot(botConfig llm.BotConfig) {
	// Create a mock bot user
	mmBot := &model.Bot{
		UserId:      "bot-user-id",
		Username:    botConfig.Name,
		DisplayName: botConfig.DisplayName,
	}

	// Create the bot instance
	bot := bots.NewBot(botConfig, mmBot)

	// Set the bot directly for testing
	e.bots.SetBotsForTesting([]*bots.Bot{bot})
}

func SetupTestEnvironment(t *testing.T) *TestEnvironment {
	mockAPI := &plugintest.API{}
	noopMetrics := &metrics.NoopMetrics{}

	client := pluginapi.NewClient(mockAPI, nil)

	// Create test bots instance
	testBots := createTestBots(mockAPI, client)

	// Create minimal conversations service for testing
	conversationsService := &conversations.Conversations{}

	api := New(testBots, conversationsService, nil, nil, nil, client, noopMetrics, nil, &testConfigImpl{}, nil, nil, nil, nil, nil, nil, &mockMCPClientManager{})

	return &TestEnvironment{
		api:     api,
		mockAPI: mockAPI,
		bots:    testBots,
	}
}

func TestPostRouter(t *testing.T) {
	// This just makes gin not output a whole bunch of debug stuff.
	// maybe pipe this to the test log?
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	for urlName, url := range map[string]string{
		"react":                   "/post/postid/react",
		"summarize":               "/post/postid/analyze",
		"transcribe":              "/post/postid/transcribe/file/fileid",
		"summarize_transcription": "/post/postid/summarize_transcription",
		"stop":                    "/post/postid/stop",
		"regenerate":              "/post/postid/regenerate",
	} {
		for name, test := range map[string]struct {
			request        *http.Request
			expectedStatus int
			botconfig      llm.BotConfig
			envSetup       func(e *TestEnvironment)
		}{
			"no permission to channel": {
				request:        httptest.NewRequest(http.MethodPost, url, nil),
				expectedStatus: http.StatusForbidden,
				envSetup: func(e *TestEnvironment) {
					e.mockAPI.On("GetChannel", "channelid").Return(&model.Channel{
						Id:     "channelid",
						Type:   model.ChannelTypeOpen,
						TeamId: "teamid",
					}, nil)
					e.mockAPI.On("HasPermissionToChannel", "userid", "channelid", model.PermissionReadChannel).Return(false)
				},
			},
			"user not allowed": {
				request:        httptest.NewRequest(http.MethodPost, url, nil),
				expectedStatus: http.StatusForbidden,
				botconfig: llm.BotConfig{
					UserAccessLevel: llm.UserAccessLevelBlock,
					UserIDs:         []string{"userid"},
				},
				envSetup: func(e *TestEnvironment) {
					e.mockAPI.On("GetChannel", "channelid").Return(&model.Channel{
						Id:     "channelid",
						Type:   model.ChannelTypeOpen,
						TeamId: "teamid",
					}, nil)
					e.mockAPI.On("HasPermissionToChannel", "userid", "channelid", model.PermissionReadChannel).Return(true)
				},
			},
		} {
			t.Run(urlName+" "+name, func(t *testing.T) {
				e := SetupTestEnvironment(t)
				defer e.Cleanup(t)

				test.botconfig.Name = "permtest"

				e.setupTestBot(test.botconfig)

				e.mockAPI.On("GetPost", "postid").Return(&model.Post{
					ChannelId: "channelid",
				}, nil)
				e.mockAPI.On("LogError", mock.Anything).Maybe()

				test.envSetup(e)

				test.request.Header.Add("Mattermost-User-ID", "userid")
				recorder := httptest.NewRecorder()
				e.api.ServeHTTP(&plugin.Context{}, recorder, test.request)
				resp := recorder.Result()
				require.Equal(t, test.expectedStatus, resp.StatusCode)
			})
		}
	}
}

func TestAdminRouter(t *testing.T) {
	// This just makes gin not output a whole bunch of debug stuff.
	// maybe pipe this to the test log?
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	for urlName, url := range map[string]string{} {
		for name, test := range map[string]struct {
			request        *http.Request
			expectedStatus int
			envSetup       func(e *TestEnvironment)
		}{
			"only admins": {
				request:        httptest.NewRequest(http.MethodGet, url, nil),
				expectedStatus: http.StatusForbidden,
				envSetup: func(e *TestEnvironment) {
					e.mockAPI.On("HasPermissionTo", "userid", model.PermissionManageSystem).Return(false)
				},
			},
		} {
			t.Run(urlName+" "+name, func(t *testing.T) {
				e := SetupTestEnvironment(t)
				defer e.Cleanup(t)

				e.mockAPI.On("LogError", mock.Anything).Maybe()

				test.envSetup(e)

				test.request.Header.Add("Mattermost-User-ID", "userid")
				recorder := httptest.NewRecorder()
				e.api.ServeHTTP(&plugin.Context{}, recorder, test.request)
				resp := recorder.Result()
				require.Equal(t, test.expectedStatus, resp.StatusCode)
			})
		}
	}
}

func TestEnforceEmptyBody(t *testing.T) {
	// This just makes gin not output a whole bunch of debug stuff.
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	tests := []struct {
		name          string
		requestBody   string
		expectedError bool
	}{
		{
			name:          "empty body",
			requestBody:   "",
			expectedError: false,
		},
		{
			name:          "non-empty body",
			requestBody:   "some content",
			expectedError: true,
		},
		{
			name:          "whitespace only",
			requestBody:   "   \n\t",
			expectedError: true,
		},
		{
			name:          "json object",
			requestBody:   `{"key": "value"}`,
			expectedError: true,
		},
		{
			name:          "empty json object",
			requestBody:   `{}`,
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := SetupTestEnvironment(t)
			defer e.Cleanup(t)

			// Create a test context with the specified request body
			w := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(w)

			// Create request with the test body
			bodyReader := strings.NewReader(test.requestBody)
			req, err := http.NewRequest("POST", "/test", bodyReader)
			require.NoError(t, err)

			ctx.Request = req

			// Test the enforceEmptyBody function
			err = e.api.enforceEmptyBody(ctx)

			if test.expectedError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "request body must be empty")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestEmptyBodyCheckerInApi tests the API endpoints that use enforceEmptyBody
func TestEmptyBodyCheckerInApi(t *testing.T) {
	// This just makes gin not output a whole bunch of debug stuff.
	// maybe pipe this to the test log?
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	for urlName, url := range map[string]string{
		"react":                   "/post/postid/react?botUsername=thebot",
		"transcribe file":         "/post/postid/transcribe/file/fileid?botUsername=thebot",
		"summarize transcription": "/post/postid/summarize_transcription?botUsername=thebot",
		"regen":                   "/post/postid/regenerate",
		"postback summary":        "/post/postid/postback_summary",
		"reindex":                 "/admin/reindex",
		"cancel":                  "/admin/reindex/cancel",
	} {
		t.Run(urlName, func(t *testing.T) {
			e := SetupTestEnvironment(t)
			defer e.Cleanup(t)

			e.mockAPI.On("LogError", "request body must be empty")
			e.mockAPI.On("GetPost", mock.Anything).Return(&model.Post{}, nil).Maybe()
			e.mockAPI.On("GetChannel", mock.Anything).Return(&model.Channel{}, nil).Maybe()
			e.mockAPI.On("HasPermissionToChannel", mock.Anything, mock.Anything, model.PermissionReadChannel).Return(true).Maybe()
			e.mockAPI.On("HasPermissionTo", mock.Anything, model.PermissionManageSystem).Return(true).Maybe()

			e.bots.SetBotsForTesting([]*bots.Bot{bots.NewBot(llm.BotConfig{Name: "thebot"}, nil)})

			request := httptest.NewRequest(http.MethodPost, url, strings.NewReader("non-empty body"))
			request.Header.Add("Mattermost-User-ID", "userid")
			recorder := httptest.NewRecorder()
			e.api.ServeHTTP(&plugin.Context{}, recorder, request)
			resp := recorder.Result()
			require.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestChannelRouter(t *testing.T) {
	// This just makes gin not output a whole bunch of debug stuff.
	// maybe pipe this to the test log?
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	for urlName, url := range map[string]string{
		"summarize since": "/channel/channelid/interval",
	} {
		for name, test := range map[string]struct {
			request        *http.Request
			expectedStatus int
			botconfig      llm.BotConfig
			envSetup       func(e *TestEnvironment)
		}{
			"test no permission to channel": {
				request:        httptest.NewRequest(http.MethodPost, url, nil),
				expectedStatus: http.StatusForbidden,
				envSetup: func(e *TestEnvironment) {
					e.mockAPI.On("GetChannel", "channelid").Return(&model.Channel{
						Id:     "channelid",
						Type:   model.ChannelTypeOpen,
						TeamId: "teamid",
					}, nil)
					e.mockAPI.On("HasPermissionToChannel", "userid", "channelid", model.PermissionReadChannel).Return(false)
				},
			},
			"test user not allowed": {
				request:        httptest.NewRequest(http.MethodPost, url, nil),
				expectedStatus: http.StatusForbidden,
				botconfig: llm.BotConfig{
					UserAccessLevel: llm.UserAccessLevelBlock,
					UserIDs:         []string{"userid"},
				},
				envSetup: func(e *TestEnvironment) {
					e.mockAPI.On("GetChannel", "channelid").Return(&model.Channel{
						Id:     "channelid",
						Type:   model.ChannelTypeOpen,
						TeamId: "teamid",
					}, nil)
					e.mockAPI.On("HasPermissionToChannel", "userid", "channelid", model.PermissionReadChannel).Return(true)
				},
			},
		} {
			t.Run(urlName+" "+name, func(t *testing.T) {
				e := SetupTestEnvironment(t)
				defer e.Cleanup(t)

				test.botconfig.Name = "permtest"

				e.setupTestBot(test.botconfig)

				e.mockAPI.On("LogError", mock.Anything).Maybe()

				test.envSetup(e)

				test.request.Header.Add("Mattermost-User-ID", "userid")
				recorder := httptest.NewRecorder()
				e.api.ServeHTTP(&plugin.Context{}, recorder, test.request)
				resp := recorder.Result()
				require.Equal(t, test.expectedStatus, resp.StatusCode)
			})
		}
	}
}

func TestHandleGetAIBots(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	tests := []struct {
		name                  string
		searchService         *search.Search
		expectedSearchEnabled bool
		expectedStatus        int
		envSetup              func(e *TestEnvironment)
	}{
		{
			name:                  "search enabled - non-nil service with non-nil embedding search",
			searchService:         search.New(mocks.NewMockEmbeddingSearch(t), nil, nil, nil, nil),
			expectedSearchEnabled: true,
			expectedStatus:        http.StatusOK,
			envSetup: func(e *TestEnvironment) {
				e.mockAPI.On("GetChannelByName", "", mock.AnythingOfType("string"), false).Return(nil, &model.AppError{})
			},
		},
		{
			name:                  "search disabled - non-nil service with nil embedding search",
			searchService:         search.New(nil, nil, nil, nil, nil),
			expectedSearchEnabled: false,
			expectedStatus:        http.StatusOK,
			envSetup: func(e *TestEnvironment) {
				e.mockAPI.On("GetChannelByName", "", mock.AnythingOfType("string"), false).Return(nil, &model.AppError{})
			},
		},
		{
			name:                  "no search service - nil service",
			searchService:         nil,
			expectedSearchEnabled: false,
			expectedStatus:        http.StatusOK,
			envSetup: func(e *TestEnvironment) {
				e.mockAPI.On("GetChannelByName", "", mock.AnythingOfType("string"), false).Return(nil, &model.AppError{})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			e := SetupTestEnvironment(t)
			defer e.Cleanup(t)

			// Override the search service for this test
			e.api.searchService = test.searchService

			// Setup a test bot
			e.setupTestBot(llm.BotConfig{
				Name:        "test-bot",
				DisplayName: "Test Bot",
			})

			// Setup mock expectations
			test.envSetup(e)
			e.mockAPI.On("LogError", mock.Anything).Maybe()

			// Create request
			request := httptest.NewRequest(http.MethodGet, "/ai_bots", nil)
			request.Header.Add("Mattermost-User-ID", "userid")

			// Execute request
			recorder := httptest.NewRecorder()
			e.api.ServeHTTP(&plugin.Context{}, recorder, request)

			// Verify status code
			resp := recorder.Result()
			require.Equal(t, test.expectedStatus, resp.StatusCode)

			// Verify response body
			if test.expectedStatus == http.StatusOK {
				var response AIBotsResponse
				err := json.NewDecoder(resp.Body).Decode(&response)
				require.NoError(t, err)
				require.Equal(t, test.expectedSearchEnabled, response.SearchEnabled, "SearchEnabled field should match expected value")
				require.NotEmpty(t, response.Bots, "Should return at least one bot")
			}
		})
	}
}
