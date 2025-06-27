// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mattermost/mattermost-plugin-ai/embeddings"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/search"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEmbeddingSearch is a mock implementation of embeddings.EmbeddingSearch for testing
type mockEmbeddingSearch struct{}

func (m *mockEmbeddingSearch) Store(ctx context.Context, docs []embeddings.PostDocument) error {
	return nil
}

func (m *mockEmbeddingSearch) Search(ctx context.Context, query string, opts embeddings.SearchOptions) ([]embeddings.SearchResult, error) {
	return nil, nil
}

func (m *mockEmbeddingSearch) Delete(ctx context.Context, postIDs []string) error {
	return nil
}

func (m *mockEmbeddingSearch) Clear(ctx context.Context) error {
	return nil
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
			searchService:         search.New(&mockEmbeddingSearch{}, nil, nil, nil, nil),
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