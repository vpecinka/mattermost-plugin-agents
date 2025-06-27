// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"bytes"
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
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEmbeddingSearch is a mock implementation for testing
type mockEmbeddingSearchAPI struct{}

func (m *mockEmbeddingSearchAPI) Store(ctx context.Context, docs []embeddings.PostDocument) error {
	return nil
}

func (m *mockEmbeddingSearchAPI) Search(ctx context.Context, query string, opts embeddings.SearchOptions) ([]embeddings.SearchResult, error) {
	return nil, nil
}

func (m *mockEmbeddingSearchAPI) Delete(ctx context.Context, postIDs []string) error {
	return nil
}

func (m *mockEmbeddingSearchAPI) Clear(ctx context.Context) error {
	return nil
}

func TestHandleRunSearch(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	tests := []struct {
		name           string
		searchService  *search.Search
		requestBody    SearchRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:          "search succeeds - service enabled",
			searchService: search.New(&mockEmbeddingSearchAPI{}, nil, nil, nil, nil),
			requestBody: SearchRequest{
				Query:      "test query",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:          "search fails - service disabled",
			searchService: search.New(nil, nil, nil, nil, nil),
			requestBody: SearchRequest{
				Query:      "test query",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:          "search fails - no service",
			searchService: nil,
			requestBody: SearchRequest{
				Query:      "test query",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:          "search fails - empty query",
			searchService: search.New(&mockEmbeddingSearchAPI{}, nil, nil, nil, nil),
			requestBody: SearchRequest{
				Query:      "",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
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
			e.mockAPI.On("LogError", mock.Anything).Maybe()

			// Create request body
			bodyBytes, err := json.Marshal(test.requestBody)
			require.NoError(t, err)

			// Create request
			request := httptest.NewRequest(http.MethodPost, "/search/run?botUsername=test-bot", bytes.NewReader(bodyBytes))
			request.Header.Add("Mattermost-User-ID", "userid")
			request.Header.Set("Content-Type", "application/json")

			// Execute request
			recorder := httptest.NewRecorder()
			e.api.ServeHTTP(&plugin.Context{}, recorder, request)

			// Verify status code
			resp := recorder.Result()
			require.Equal(t, test.expectedStatus, resp.StatusCode)
		})
	}
}

func TestHandleSearchQuery(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	gin.DefaultWriter = io.Discard

	tests := []struct {
		name           string
		searchService  *search.Search
		requestBody    SearchRequest
		expectedStatus int
		expectError    bool
	}{
		{
			name:          "search query succeeds - service enabled",
			searchService: search.New(&mockEmbeddingSearchAPI{}, nil, nil, nil, nil),
			requestBody: SearchRequest{
				Query:      "test query",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:          "search query fails - service disabled",
			searchService: search.New(nil, nil, nil, nil, nil),
			requestBody: SearchRequest{
				Query:      "test query",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:          "search query fails - no service",
			searchService: nil,
			requestBody: SearchRequest{
				Query:      "test query",
				TeamID:     "team123",
				ChannelID:  "channel123",
				MaxResults: 10,
			},
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
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
			e.mockAPI.On("LogError", mock.Anything).Maybe()

			// Create request body
			bodyBytes, err := json.Marshal(test.requestBody)
			require.NoError(t, err)

			// Create request
			request := httptest.NewRequest(http.MethodPost, "/search?botUsername=test-bot", bytes.NewReader(bodyBytes))
			request.Header.Add("Mattermost-User-ID", "userid")
			request.Header.Set("Content-Type", "application/json")

			// Execute request
			recorder := httptest.NewRecorder()
			e.api.ServeHTTP(&plugin.Context{}, recorder, request)

			// Verify status code
			resp := recorder.Result()
			require.Equal(t, test.expectedStatus, resp.StatusCode)
		})
	}
}