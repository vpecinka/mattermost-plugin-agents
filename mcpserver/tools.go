// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"
	"strconv"
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

// searchUsers implements the search_users tool
func (p *MattermostToolProvider) searchUsers(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	term, ok := arguments["term"].(string)
	if !ok {
		return nil, fmt.Errorf("term is required and must be a string")
	}

	// Extract optional limit
	limit := 20
	if val, exists := arguments["limit"]; exists {
		if num, ok := val.(float64); ok {
			limit = int(num)
		} else if str, ok := val.(string); ok {
			if parsed, err := strconv.Atoi(str); err == nil {
				limit = parsed
			}
		}
	}

	// Validate limit
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 1
	}

	// Search for users
	users, _, err := client.SearchUsers(context.Background(), &model.UserSearch{
		Term: term,
	})
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error searching users: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Limit results
	if len(users) > limit {
		users = users[:limit]
	}

	if len(users) == 0 {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("No users found matching term: %s", term),
			}},
		}, nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d users matching '%s':\n", len(users), term))
	for i, user := range users {
		result.WriteString(fmt.Sprintf("  %d. %s (%s) - %s %s <%s>\n",
			i+1, user.Username, user.Id, user.FirstName, user.LastName, user.Email))
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

// getChannelMembers implements the get_channel_members tool
func (p *MattermostToolProvider) getChannelMembers(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	channelID, ok := arguments["channel_id"].(string)
	if !ok {
		return nil, fmt.Errorf("channel_id is required and must be a string")
	}

	// Get channel members
	members, _, err := client.GetChannelMembers(context.Background(), channelID, 0, 200, "")
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting channel members: %v", err),
			}},
			IsError: true,
		}, nil
	}

	if len(members) == 0 {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("No members found in channel %s", channelID),
			}},
		}, nil
	}

	// Get user details for each member
	var users []*model.User
	for _, member := range members {
		user, _, err := client.GetUser(context.Background(), member.UserId, "")
		if err != nil {
			continue // Skip users we can't fetch
		}
		users = append(users, user)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d members in channel %s:\n", len(users), channelID))
	for i, user := range users {
		result.WriteString(fmt.Sprintf("  %d. %s (%s) - %s %s <%s>\n",
			i+1, user.Username, user.Id, user.FirstName, user.LastName, user.Email))
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}

// getTeamMembers implements the get_team_members tool
func (p *MattermostToolProvider) getTeamMembers(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	teamID, ok := arguments["team_id"].(string)
	if !ok {
		return nil, fmt.Errorf("team_id is required and must be a string")
	}

	// Get team members
	members, _, err := client.GetTeamMembers(context.Background(), teamID, 0, 200, "")
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error getting team members: %v", err),
			}},
			IsError: true,
		}, nil
	}

	if len(members) == 0 {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("No members found in team %s", teamID),
			}},
		}, nil
	}

	// Get user details for each member
	var users []*model.User
	for _, member := range members {
		user, _, err := client.GetUser(context.Background(), member.UserId, "")
		if err != nil {
			continue // Skip users we can't fetch
		}
		users = append(users, user)
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d members in team %s:\n", len(users), teamID))
	for i, user := range users {
		result.WriteString(fmt.Sprintf("  %d. %s (%s) - %s %s <%s>\n",
			i+1, user.Username, user.Id, user.FirstName, user.LastName, user.Email))
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: result.String(),
		}},
	}, nil
}
