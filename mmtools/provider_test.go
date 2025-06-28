// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mmtools

import (
	"errors"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost-plugin-ai/bots"
	"github.com/mattermost/mattermost-plugin-ai/embeddings"
	"github.com/mattermost/mattermost-plugin-ai/embeddings/mocks"
	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost-plugin-ai/search"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMMToolProvider_GetTools(t *testing.T) {
	tests := []struct {
		name                      string
		searchService             *search.Search
		isDM                      bool
		expectedSearchToolPresent bool
	}{
		{
			name:                      "search tool available - search enabled in DM",
			searchService:             search.New(mocks.NewMockEmbeddingSearch(t), nil, nil, nil, nil),
			isDM:                      true,
			expectedSearchToolPresent: true,
		},
		{
			name:                      "search tool not available - search disabled in DM",
			searchService:             search.New(nil, nil, nil, nil, nil),
			isDM:                      true,
			expectedSearchToolPresent: false,
		},
		{
			name:                      "search tool not available - no search service in DM",
			searchService:             nil,
			isDM:                      true,
			expectedSearchToolPresent: false,
		},
		{
			name:                      "search tool not available - not in DM (channel context)",
			searchService:             search.New(mocks.NewMockEmbeddingSearch(t), nil, nil, nil, nil),
			isDM:                      false,
			expectedSearchToolPresent: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create tool provider
			provider := NewMMToolProvider(nil, test.searchService, &http.Client{})

			// Create a mock bot
			bot := &bots.Bot{}

			// Get tools
			tools := provider.GetTools(test.isDM, bot)

			// Check if SearchServer tool is present
			searchToolFound := false
			for _, tool := range tools {
				if tool.Name == "SearchServer" {
					searchToolFound = true
					break
				}
			}

			require.Equal(t, test.expectedSearchToolPresent, searchToolFound,
				"SearchServer tool presence should match expected value")
		})
	}
}

func TestMMToolProvider_toolSearchServer(t *testing.T) {
	tests := []struct {
		name          string
		searchService *search.Search
		searchTerm    string
		expectError   bool
		expectedMsg   string
	}{
		{
			name: "search succeeds - service enabled",
			searchService: func() *search.Search {
				mockEmbedding := mocks.NewMockEmbeddingSearch(t)
				mockEmbedding.On("Search", mock.Anything, "test search term", mock.Anything).Return([]embeddings.SearchResult{}, nil)
				return search.New(mockEmbedding, nil, nil, nil, nil)
			}(),
			searchTerm:  "test search term",
			expectError: false,
			expectedMsg: "No relevant messages found.", // mock returns empty results
		},
		{
			name:          "search fails - service disabled",
			searchService: search.New(nil, nil, nil, nil, nil),
			searchTerm:    "test search term",
			expectError:   true,
			expectedMsg:   "search functionality is not configured",
		},
		{
			name:          "search fails - no service",
			searchService: nil,
			searchTerm:    "test search term",
			expectError:   true,
			expectedMsg:   "search functionality is not configured",
		},
		{
			name: "search fails - term too short",
			searchService: func() *search.Search {
				mockEmbedding := mocks.NewMockEmbeddingSearch(t)
				return search.New(mockEmbedding, nil, nil, nil, nil)
			}(),
			searchTerm:  "hi",
			expectError: true,
			expectedMsg: "search term too short",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Create tool provider
			provider := NewMMToolProvider(nil, test.searchService, &http.Client{})

			// Create mock LLM context
			llmContext := &llm.Context{
				RequestingUser: &model.User{Id: "user123"},
			}

			// Create argument getter
			argsGetter := func(args interface{}) error {
				if searchArgs, ok := args.(*SearchServerArgs); ok {
					searchArgs.Term = test.searchTerm
					return nil
				}
				return errors.New("invalid args")
			}

			// Execute the tool
			result, err := provider.toolSearchServer(llmContext, argsGetter)

			// Verify results
			if test.expectError {
				require.Error(t, err)
				require.Equal(t, test.expectedMsg, result)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expectedMsg, result)
			}
		})
	}
}
