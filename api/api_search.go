// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mattermost/mattermost-plugin-ai/bots"
)

// SearchRequest represents a search query request from the API
type SearchRequest struct {
	Query      string `json:"query"`
	TeamID     string `json:"teamId"`
	ChannelID  string `json:"channelId"`
	MaxResults int    `json:"maxResults"`
}

func (a *API) handleRunSearch(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	bot := c.MustGet(ContextBotKey).(*bots.Bot)

	if a.searchService == nil || !a.searchService.Enabled() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("search functionality is not configured"))
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}

	if req.Query == "" {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("query cannot be empty"))
		return
	}

	result, err := a.searchService.RunSearch(c.Request.Context(), userID, bot, req.Query, req.TeamID, req.ChannelID, req.MaxResults)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (a *API) handleSearchQuery(c *gin.Context) {
	userID := c.GetHeader("Mattermost-User-Id")
	bot := c.MustGet(ContextBotKey).(*bots.Bot)

	if a.searchService == nil || !a.searchService.Enabled() {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("search functionality is not configured"))
		return
	}

	var req SearchRequest
	if err := json.NewDecoder(c.Request.Body).Decode(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, fmt.Errorf("invalid request: %w", err))
		return
	}

	response, err := a.searchService.SearchQuery(c.Request.Context(), userID, bot, req.Query, req.TeamID, req.ChannelID, req.MaxResults)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, response)
}
