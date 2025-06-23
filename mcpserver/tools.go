// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// MattermostToolProvider implements Mattermost operations for MCP tools
type MattermostToolProvider struct {
	authProvider AuthenticationProvider
	logger       mlog.LoggerIFace
}

// NewMattermostToolProvider creates a new Mattermost tool provider
func NewMattermostToolProvider(authProvider AuthenticationProvider, logger mlog.LoggerIFace) *MattermostToolProvider {
	return &MattermostToolProvider{
		authProvider: authProvider,
		logger:       logger,
	}
}

// GetTools returns the available MCP tools for Mattermost
func (p *MattermostToolProvider) GetTools() []Tool {
	return []Tool{
		{
			Name:        "read_post",
			Description: "Read a specific post and its thread from Mattermost",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"post_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the post to read",
					},
					"include_thread": map[string]interface{}{
						"type":        "boolean",
						"description": "Whether to include the entire thread (default: true)",
					},
				},
				"required": []string{"post_id"},
			},
		},
		{
			Name:        "read_channel",
			Description: "Read recent posts from a Mattermost channel",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the channel to read from",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of posts to retrieve (default: 20, max: 100)",
						"minimum":     1,
						"maximum":     100,
					},
					"since": map[string]interface{}{
						"type":        "string",
						"description": "Only get posts since this timestamp (ISO 8601 format)",
					},
				},
				"required": []string{"channel_id"},
			},
		},
		{
			Name:        "search_posts",
			Description: "Search for posts in Mattermost",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional team ID to limit search scope",
					},
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional channel ID to limit search to a specific channel",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Number of results to return (default: 20, max: 100)",
						"minimum":     1,
						"maximum":     100,
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "create_post",
			Description: "Create a new post in Mattermost",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the channel to post in",
					},
					"message": map[string]interface{}{
						"type":        "string",
						"description": "The message content",
					},
					"root_id": map[string]interface{}{
						"type":        "string",
						"description": "Optional root post ID for replies",
					},
					"props": map[string]interface{}{
						"type":        "object",
						"description": "Optional post properties",
					},
				},
				"required": []string{"channel_id", "message"},
			},
		},
		{
			Name:        "create_channel",
			Description: "Create a new channel in Mattermost",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "The channel name (URL-friendly)",
					},
					"display_name": map[string]interface{}{
						"type":        "string",
						"description": "The channel display name",
					},
					"type": map[string]interface{}{
						"type":        "string",
						"description": "Channel type: 'O' for public, 'P' for private",
						"enum":        []string{"O", "P"},
					},
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "The team ID where the channel will be created",
					},
					"purpose": map[string]interface{}{
						"type":        "string",
						"description": "Optional channel purpose",
					},
					"header": map[string]interface{}{
						"type":        "string",
						"description": "Optional channel header",
					},
				},
				"required": []string{"name", "display_name", "type", "team_id"},
			},
		},
		{
			Name:        "get_channel_info",
			Description: "Get information about a channel",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"channel_id": map[string]interface{}{
						"type":        "string",
						"description": "The ID of the channel",
					},
					"channel_name": map[string]interface{}{
						"type":        "string",
						"description": "The name of the channel (if ID not provided)",
					},
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID (required if using channel_name)",
					},
				},
				"anyOf": []map[string]interface{}{
					{"required": []string{"channel_id"}},
					{"required": []string{"channel_name", "team_id"}},
				},
			},
		},
	}
}

// ExecuteTool executes a specific tool with the given arguments
func (p *MattermostToolProvider) ExecuteTool(ctx context.Context, toolName string, arguments map[string]interface{}) (*ToolResult, error) {
	// Get user ID from context
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("user ID not found in context")
	}

	// Get token from context
	token, tokenOk := ctx.Value(TokenKey).(string)
	if !tokenOk {
		// For stdio mode, token might not be in context - use empty string
		token = ""
	}

	// Get authenticated client for this user
	client, err := p.authProvider.GetMattermostClient(ctx, userID, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated client: %w", err)
	}

	switch toolName {
	case "read_post":
		return p.readPost(ctx, client, arguments)
	case "read_channel":
		return p.readChannel(ctx, client, arguments)
	case "search_posts":
		return p.searchPosts(ctx, client, arguments)
	case "create_post":
		return p.createPost(ctx, client, arguments)
	case "create_channel":
		return p.createChannel(ctx, client, arguments)
	case "get_channel_info":
		return p.getChannelInfo(ctx, client, arguments)
	case "get_team_info":
		return p.getTeamInfo(ctx, client, arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

// readPost implements the read_post tool
func (p *MattermostToolProvider) readPost(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	postID, ok := arguments["post_id"].(string)
	if !ok {
		return nil, fmt.Errorf("post_id is required and must be a string")
	}

	includeThread := true
	if val, exists := arguments["include_thread"]; exists {
		if b, includeOk := val.(bool); includeOk {
			includeThread = b
		}
	}

	// client is already *model.Client4 from function signature

	var posts []*model.Post

	if includeThread {
		postList, _, err := client.GetPostThread(context.Background(), postID, "", false)
		if err != nil {
			return &ToolResult{
				Content: []Content{{
					Type: "text",
					Text: fmt.Sprintf("Error reading post thread: %v", err),
				}},
				IsError: true,
			}, nil
		}

		// Convert PostList to ordered slice
		posts = postList.ToSlice()
	} else {
		post, _, err := client.GetPost(context.Background(), postID, "")
		if err != nil {
			return &ToolResult{
				Content: []Content{{
					Type: "text",
					Text: fmt.Sprintf("Error reading post: %v", err),
				}},
				IsError: true,
			}, nil
		}
		posts = []*model.Post{post}
	}

	// Format the response
	result := strings.Builder{}

	// Get channel info for the first post
	if len(posts) > 0 {
		channel, _, err := client.GetChannel(context.Background(), posts[0].ChannelId, "")
		if err == nil {
			result.WriteString(fmt.Sprintf("## Channel: %s\n\n", channel.DisplayName))
		}
	}

	for i, post := range posts {
		// Get user info
		user, _, err := client.GetUser(context.Background(), post.UserId, "")
		username := "Unknown User"
		if err == nil {
			username = user.Username
		}

		// Format timestamp
		timestamp := time.Unix(post.CreateAt/1000, 0).Format("2006-01-02 15:04:05")

		// Format post
		if i == 0 || post.RootId == "" {
			result.WriteString(fmt.Sprintf("### Post by @%s at %s\n", username, timestamp))
		} else {
			result.WriteString(fmt.Sprintf("#### Reply by @%s at %s\n", username, timestamp))
		}
		result.WriteString(fmt.Sprintf("**Post ID:** %s\n\n", post.Id))
		result.WriteString(post.Message)
		result.WriteString("\n\n---\n\n")
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

// readChannel implements the read_channel tool
func (p *MattermostToolProvider) readChannel(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	channelID, ok := arguments["channel_id"].(string)
	if !ok {
		return nil, fmt.Errorf("channel_id is required and must be a string")
	}

	limit := 20
	if val, exists := arguments["limit"]; exists {
		if l, limitOk := val.(float64); limitOk {
			limit = int(l)
		} else if l, limitIntOk := val.(int); limitIntOk {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}

	// Get channel info
	channel, _, err := client.GetChannel(context.Background(), channelID, "")
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting channel: %v", err),
			}},
			IsError: true,
		}, nil
	}

	var postList *model.PostList

	// Check if since parameter is provided
	if sinceStr, exists := arguments["since"]; exists {
		if since, sinceOk := sinceStr.(string); sinceOk {
			// Parse ISO 8601 timestamp
			sinceTime, parseErr := time.Parse(time.RFC3339, since)
			if parseErr != nil {
				return &ToolResult{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("Error parsing since timestamp: %v", parseErr),
					}},
					IsError: true,
				}, nil
			}

			sinceMs := sinceTime.UnixMilli()
			postList, _, err = client.GetPostsSince(context.Background(), channelID, sinceMs, false)
		} else {
			// Get recent posts
			postList, _, err = client.GetPostsBefore(context.Background(), channelID, "", 0, limit, "", false, false)
		}
	} else {
		// Get recent posts
		postList, _, err = client.GetPostsBefore(context.Background(), channelID, "", 0, limit, "", false, false)
	}

	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting posts: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format the response
	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("## Channel: %s\n", channel.DisplayName))
	result.WriteString(fmt.Sprintf("**Channel ID:** %s\n", channel.Id))
	result.WriteString(fmt.Sprintf("**Posts retrieved:** %d\n\n", len(postList.Posts)))

	// Convert to ordered slice
	posts := postList.ToSlice()

	for _, post := range posts {
		// Get user info
		user, _, err := client.GetUser(context.Background(), post.UserId, "")
		username := "Unknown User"
		if err == nil {
			username = user.Username
		}

		// Format timestamp
		timestamp := time.Unix(post.CreateAt/1000, 0).Format("2006-01-02 15:04:05")

		result.WriteString(fmt.Sprintf("### @%s - %s\n", username, timestamp))
		result.WriteString(fmt.Sprintf("**Post ID:** %s\n\n", post.Id))
		result.WriteString(post.Message)
		result.WriteString("\n\n---\n\n")
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

// searchPosts implements the search_posts tool
func (p *MattermostToolProvider) searchPosts(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	query, ok := arguments["query"].(string)
	if !ok {
		return nil, fmt.Errorf("query is required and must be a string")
	}

	teamID := ""
	if val, exists := arguments["team_id"]; exists {
		if s, teamOk := val.(string); teamOk {
			teamID = s
		}
	}

	// Perform search
	searchParams := &model.SearchParameter{
		Terms: &query,
	}
	postList, _, err := client.SearchPostsWithParams(context.Background(), teamID, searchParams)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error searching posts: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format the response
	result := strings.Builder{}
	result.WriteString(fmt.Sprintf("## Search Results for: \"%s\"\n", query))
	result.WriteString(fmt.Sprintf("**Results found:** %d\n\n", len(postList.Posts)))

	// Convert to ordered slice and limit results
	posts := postList.ToSlice()
	limit := 20
	if val, exists := arguments["limit"]; exists {
		if l, ok := val.(float64); ok {
			limit = int(l)
		} else if l, ok := val.(int); ok {
			limit = l
		}
	}
	if limit > 100 {
		limit = 100
	}
	if len(posts) > limit {
		posts = posts[:limit]
	}

	for _, post := range posts {
		// Get user info
		user, _, err := client.GetUser(context.Background(), post.UserId, "")
		username := "Unknown User"
		if err == nil {
			username = user.Username
		}

		// Get channel info
		channel, _, err := client.GetChannel(context.Background(), post.ChannelId, "")
		channelName := "Unknown Channel"
		if err == nil {
			channelName = channel.DisplayName
		}

		// Format timestamp
		timestamp := time.Unix(post.CreateAt/1000, 0).Format("2006-01-02 15:04:05")

		result.WriteString(fmt.Sprintf("### @%s in %s - %s\n", username, channelName, timestamp))
		result.WriteString(fmt.Sprintf("**Post ID:** %s\n\n", post.Id))
		result.WriteString(post.Message)
		result.WriteString("\n\n---\n\n")
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

// createPost implements the create_post tool
func (p *MattermostToolProvider) createPost(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	channelID, ok := arguments["channel_id"].(string)
	if !ok {
		return nil, fmt.Errorf("channel_id is required and must be a string")
	}

	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required and must be a string")
	}

	// client is already *model.Client4 from function signature

	// Get user ID from context
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("user ID not found in context")
	}

	// Create post
	post := &model.Post{
		ChannelId: channelID,
		UserId:    userID,
		Message:   message,
	}

	// Set root ID if this is a reply
	if rootID, exists := arguments["root_id"]; exists {
		if s, ok := rootID.(string); ok && s != "" {
			post.RootId = s
		}
	}

	// Set props if provided
	if props, exists := arguments["props"]; exists {
		if p, ok := props.(map[string]interface{}); ok {
			post.SetProps(p)
		}
	}

	_, _, err := client.CreatePost(context.Background(), post)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating post: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Get channel info for response
	channel, _, err := client.GetChannel(context.Background(), channelID, "")
	channelName := channelID
	if err == nil {
		channelName = channel.DisplayName
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Post created successfully in channel '%s'", channelName),
		}},
	}, nil
}

// createChannel implements the create_channel tool
func (p *MattermostToolProvider) createChannel(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	name, ok := arguments["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required and must be a string")
	}

	displayName, ok := arguments["display_name"].(string)
	if !ok {
		return nil, fmt.Errorf("display_name is required and must be a string")
	}

	channelType, ok := arguments["type"].(string)
	if !ok {
		return nil, fmt.Errorf("type is required and must be a string")
	}

	teamID, ok := arguments["team_id"].(string)
	if !ok {
		return nil, fmt.Errorf("team_id is required and must be a string")
	}

	// client is already *model.Client4 from function signature

	// Get user ID from context
	userID, ok := ctx.Value(UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf("user ID not found in context")
	}

	// Create channel
	channel := &model.Channel{
		Name:        name,
		DisplayName: displayName,
		Type:        model.ChannelType(channelType),
		TeamId:      teamID,
		CreatorId:   userID,
	}

	// Set optional fields
	if purpose, exists := arguments["purpose"]; exists {
		if s, ok := purpose.(string); ok {
			channel.Purpose = s
		}
	}

	if header, exists := arguments["header"]; exists {
		if s, ok := header.(string); ok {
			channel.Header = s
		}
	}

	createdChannel, _, err := client.CreateChannel(context.Background(), channel)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating channel: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Channel '%s' (ID: %s) created successfully", createdChannel.DisplayName, createdChannel.Id),
		}},
	}, nil
}

// getChannelInfo implements the get_channel_info tool
func (p *MattermostToolProvider) getChannelInfo(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// client is already *model.Client4 from function signature
	var channel *model.Channel
	var err error

	// Get channel by ID or name
	if channelID, exists := arguments["channel_id"]; exists {
		if id, ok := channelID.(string); ok {
			channel, _, err = client.GetChannel(context.Background(), id, "")
		}
	} else if channelName, exists := arguments["channel_name"]; exists {
		if name, ok := channelName.(string); ok {
			teamID, teamExists := arguments["team_id"].(string)
			if !teamExists {
				return nil, fmt.Errorf("team_id is required when using channel_name")
			}
			channel, _, err = client.GetChannelByName(context.Background(), name, teamID, "")
		}
	} else {
		return nil, fmt.Errorf("either channel_id or channel_name must be provided")
	}

	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting channel: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Format channel info
	channelTypeStr := "Unknown"
	switch channel.Type {
	case model.ChannelTypeOpen:
		channelTypeStr = "Public"
	case model.ChannelTypePrivate:
		channelTypeStr = "Private"
	case model.ChannelTypeDirect:
		channelTypeStr = "Direct Message"
	case model.ChannelTypeGroup:
		channelTypeStr = "Group Message"
	}

	result := fmt.Sprintf(`## Channel Information

**Name:** %s
**Display Name:** %s
**ID:** %s
**Type:** %s
**Team ID:** %s
**Created:** %s
**Purpose:** %s
**Header:** %s
`,
		channel.Name,
		channel.DisplayName,
		channel.Id,
		channelTypeStr,
		channel.TeamId,
		time.Unix(channel.CreateAt/1000, 0).Format("2006-01-02 15:04:05"),
		channel.Purpose,
		channel.Header,
	)

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result,
		}},
	}, nil
}

// getTeamInfo implements the get_team_info tool
func (p *MattermostToolProvider) getTeamInfo(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// client is already *model.Client4 from function signature
	var team *model.Team
	var err error

	// Get team by ID, name, or display name
	if teamID, exists := arguments["team_id"]; exists {
		if id, ok := teamID.(string); ok && id != "" {
			team, _, err = client.GetTeam(context.Background(), id, "")
		}
	} else if teamName, exists := arguments["team_name"]; exists {
		if name, ok := teamName.(string); ok && name != "" {
			team, _, err = client.GetTeamByName(context.Background(), name, "")
		}
	} else if teamDisplayName, exists := arguments["team_display_name"]; exists {
		if displayName, ok := teamDisplayName.(string); ok && displayName != "" {
			// Use SearchTeams to find team by display name
			searchRequest := &model.TeamSearch{
				Term: displayName,
			}

			teams, _, searchErr := client.SearchTeams(context.Background(), searchRequest)
			if searchErr != nil {
				return &ToolResult{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("Error searching teams: %v", searchErr),
					}},
					IsError: true,
				}, nil
			}

			// Find exact match for display name (case-insensitive)
			for _, t := range teams {
				if strings.EqualFold(t.DisplayName, displayName) {
					team = t
					break
				}
			}

			// If no exact match, check if we found any partial matches
			if team == nil {
				if len(teams) > 0 {
					// Return the first match with a note about partial matching
					team = teams[0]
					result := fmt.Sprintf(`## Team Information (Partial Match)

**Note:** No exact match found for "%s". Showing closest match:

**Name:** %s
**Display Name:** %s
**ID:** %s
**Type:** %s
**Description:** %s
**Created:** %s
**Allow Open Invite:** %t
**Invite ID:** %s

**Other matches found:** %d
`,
						displayName,
						team.Name,
						team.DisplayName,
						team.Id,
						getTeamTypeString(team.Type),
						team.Description,
						time.Unix(team.CreateAt/1000, 0).Format("2006-01-02 15:04:05"),
						team.AllowOpenInvite,
						team.InviteId,
						len(teams),
					)

					return &ToolResult{
						Content: []Content{{
							Type: "text",
							Text: result,
						}},
					}, nil
				}
				return &ToolResult{
					Content: []Content{{
						Type: "text",
						Text: fmt.Sprintf("No teams found matching '%s'", displayName),
					}},
					IsError: true,
				}, nil
			}
		}
	} else {
		return nil, fmt.Errorf("either team_id, team_name, or team_display_name must be provided")
	}

	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting team: %v", err),
			}},
			IsError: true,
		}, nil
	}

	if team == nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: "Team not found",
			}},
			IsError: true,
		}, nil
	}

	// Format team info
	result := fmt.Sprintf(`## Team Information

**Name:** %s
**Display Name:** %s
**ID:** %s
**Type:** %s
**Description:** %s
**Created:** %s
**Allow Open Invite:** %t
**Invite ID:** %s
`,
		team.Name,
		team.DisplayName,
		team.Id,
		getTeamTypeString(team.Type),
		team.Description,
		time.Unix(team.CreateAt/1000, 0).Format("2006-01-02 15:04:05"),
		team.AllowOpenInvite,
		team.InviteId,
	)

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result,
		}},
	}, nil
}

// getTeamTypeString returns a human-readable string for the team type
func getTeamTypeString(teamType string) string {
	switch teamType {
	case model.TeamOpen:
		return "Open"
	case model.TeamInvite:
		return "Invite Only"
	default:
		return "Unknown"
	}
}
