// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package tools

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-plugin-ai/llm"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// ReadChannelArgs represents arguments for the read_channel tool
type ReadChannelArgs struct {
	ChannelID string `json:"channel_id" jsonschema_description:"The ID of the channel to read from"`
	Limit     int    `json:"limit" jsonschema_description:"Number of posts to retrieve (default: 20, max: 100)"`
	Since     string `json:"since" jsonschema_description:"Only get posts since this timestamp (ISO 8601 format)"`
}

// CreateChannelArgs represents arguments for the create_channel tool
type CreateChannelArgs struct {
	Name        string `json:"name" jsonschema_description:"The channel name (URL-friendly)"`
	DisplayName string `json:"display_name" jsonschema_description:"The channel display name"`
	Type        string `json:"type" jsonschema_description:"Channel type: 'O' for public, 'P' for private"`
	TeamID      string `json:"team_id" jsonschema_description:"The team ID where the channel will be created"`
	Purpose     string `json:"purpose" jsonschema_description:"Optional channel purpose"`
	Header      string `json:"header" jsonschema_description:"Optional channel header"`
}

// GetChannelInfoArgs represents arguments for the get_channel_info tool
type GetChannelInfoArgs struct {
	ChannelID          string `json:"channel_id" jsonschema_description:"The exact channel ID (fastest, most reliable method)"`
	ChannelDisplayName string `json:"channel_display_name" jsonschema_description:"The human-readable display name users see (e.g. 'General Discussion')"`
	ChannelName        string `json:"channel_name" jsonschema_description:"The URL-friendly channel name (e.g. 'general-discussion')"`
	TeamID             string `json:"team_id" jsonschema_description:"Team ID (required if using channel_name or channel_display_name)"`
}

// GetChannelMembersArgs represents arguments for the get_channel_members tool
type GetChannelMembersArgs struct {
	ChannelID string `json:"channel_id" jsonschema_description:"ID of the channel to get members for"`
	Limit     int    `json:"limit" jsonschema_description:"Number of members to return (default: 50, max: 200)"`
	Page      int    `json:"page" jsonschema_description:"Page number for pagination (default: 0)"`
}

// AddUserToChannelArgs represents arguments for the add_user_to_channel tool (dev mode only)
type AddUserToChannelArgs struct {
	UserID    string `json:"user_id" jsonschema_description:"ID of the user to add"`
	ChannelID string `json:"channel_id" jsonschema_description:"ID of the channel to add user to"`
}

// getChannelTools returns all channel-related tools
func (p *MattermostToolProvider) getChannelTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "read_channel",
			Description: "Read recent posts from a Mattermost channel",
			Schema:      llm.NewJSONSchemaFromStruct[ReadChannelArgs](),
			Resolver:    p.toolReadChannel,
		},
		{
			Name:        "create_channel",
			Description: "Create a new channel in Mattermost",
			Schema:      llm.NewJSONSchemaFromStruct[CreateChannelArgs](),
			Resolver:    p.toolCreateChannel,
		},
		{
			Name:        "get_channel_info",
			Description: "Get information about a channel. If you have a channel ID, use that for fastest lookup. If the user provides a human-readable name, try channel_display_name first (what users see in the UI), then channel_name (URL name) as fallback.",
			Schema:      llm.NewJSONSchemaFromStruct[GetChannelInfoArgs](),
			Resolver:    p.toolGetChannelInfo,
		},
		{
			Name:        "get_channel_members",
			Description: "Get members of a channel with pagination support",
			Schema:      llm.NewJSONSchemaFromStruct[GetChannelMembersArgs](),
			Resolver:    p.toolGetChannelMembers,
		},
	}
}

// getDevChannelTools returns development channel-related tools for MCP
func (p *MattermostToolProvider) getDevChannelTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "add_user_to_channel",
			Description: "Add a user to a channel (dev mode only)",
			Schema:      llm.NewJSONSchemaFromStruct[AddUserToChannelArgs](),
			Resolver:    p.toolAddUserToChannel,
		},
	}
}

// toolReadChannel implements the read_channel tool
func (p *MattermostToolProvider) toolReadChannel(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args ReadChannelArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool read_channel: %w", err)
	}

	// Set defaults and validate
	if args.Limit == 0 {
		args.Limit = 20
	}
	if args.Limit > 100 {
		args.Limit = 100
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	// Parse since timestamp if provided
	var since int64
	if args.Since != "" {
		parsedTime, parseErr := time.Parse(time.RFC3339, args.Since)
		if parseErr != nil {
			return "invalid since timestamp format", fmt.Errorf("invalid timestamp format: %w", parseErr)
		}
		since = parsedTime.Unix() * 1000 // Convert to milliseconds
	}

	// Get posts from the channel
	posts, _, err := client.GetPostsForChannel(ctx, args.ChannelID, 0, args.Limit, "", false, false)
	if err != nil {
		return "failed to fetch channel posts", fmt.Errorf("error fetching posts: %w", err)
	}

	// Filter by since timestamp if provided
	var filteredPosts []*model.Post
	for _, post := range posts.ToSlice() {
		if since == 0 || post.CreateAt >= since {
			filteredPosts = append(filteredPosts, post)
		}
	}

	if len(filteredPosts) == 0 {
		return "no posts found in the specified timeframe", nil
	}

	// Format the response
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Found %d posts in channel:\n\n", len(filteredPosts)))

	for i, post := range filteredPosts {
		// Get user info for the post
		user, _, err := client.GetUser(ctx, post.UserId, "")
		if err != nil {
			p.logger.Warn("failed to get user for post", mlog.String("user_id", post.UserId), mlog.Err(err))
			result.WriteString(fmt.Sprintf("**Post %d** by Unknown User:\n", i+1))
		} else {
			result.WriteString(fmt.Sprintf("**Post %d** by %s:\n", i+1, user.Username))
		}

		result.WriteString(fmt.Sprintf("Post ID: %s\n", post.Id))
		result.WriteString(fmt.Sprintf("%s\n\n", post.Message))
	}

	return result.String(), nil
}

// toolCreateChannel implements the create_channel tool
func (p *MattermostToolProvider) toolCreateChannel(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args CreateChannelArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool create_channel: %w", err)
	}

	// Validate required fields
	if args.Name == "" {
		return "name is required", fmt.Errorf("name cannot be empty")
	}
	if args.DisplayName == "" {
		return "display_name is required", fmt.Errorf("display_name cannot be empty")
	}
	if args.Type == "" {
		return "type is required", fmt.Errorf("type cannot be empty")
	}
	if args.TeamID == "" {
		return "team_id is required", fmt.Errorf("team_id cannot be empty")
	}

	// Validate channel type
	if args.Type != "O" && args.Type != "P" {
		return "type must be 'O' for public or 'P' for private", fmt.Errorf("invalid channel type: %s", args.Type)
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	// Create the channel
	channel := &model.Channel{
		TeamId:      args.TeamID,
		Type:        model.ChannelType(args.Type),
		DisplayName: args.DisplayName,
		Name:        args.Name,
		Purpose:     args.Purpose,
		Header:      args.Header,
	}

	createdChannel, _, err := client.CreateChannel(ctx, channel)
	if err != nil {
		return "failed to create channel", fmt.Errorf("error creating channel: %w", err)
	}

	return fmt.Sprintf("Successfully created channel '%s' with ID: %s", createdChannel.DisplayName, createdChannel.Id), nil
}

// toolGetChannelInfo implements the get_channel_info tool
func (p *MattermostToolProvider) toolGetChannelInfo(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args GetChannelInfoArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool get_channel_info: %w", err)
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	var channel *model.Channel

	// Try different lookup methods based on provided parameters
	switch {
	case args.ChannelID != "":
		// Direct ID lookup - fastest method
		channel, _, err = client.GetChannel(ctx, args.ChannelID, "")
		if err != nil {
			return "channel not found by ID", fmt.Errorf("error fetching channel by ID: %w", err)
		}
	case args.ChannelDisplayName != "" && args.TeamID != "":
		// Lookup by display name
		// Get current user ID for the API call
		user, _, userErr := client.GetMe(ctx, "")
		if userErr != nil {
			return "failed to get current user", fmt.Errorf("error getting current user: %w", userErr)
		}

		channels, _, channelErr := client.GetChannelsForTeamForUser(ctx, args.TeamID, user.Id, false, "")
		if channelErr != nil {
			return "failed to fetch team channels", fmt.Errorf("error fetching team channels: %w", channelErr)
		}

		for _, ch := range channels {
			if ch.DisplayName == args.ChannelDisplayName {
				channel = ch
				break
			}
		}

		if channel == nil {
			return "channel not found by display name", fmt.Errorf("no channel found with display name: %s", args.ChannelDisplayName)
		}
	case args.ChannelName != "" && args.TeamID != "":
		// Lookup by name
		channel, _, err = client.GetChannelByName(ctx, args.ChannelName, args.TeamID, "")
		if err != nil {
			return "channel not found by name", fmt.Errorf("error fetching channel by name: %w", err)
		}
	default:
		return "either channel_id or (channel_name/channel_display_name + team_id) must be provided", fmt.Errorf("insufficient parameters for channel lookup")
	}

	// Format the response
	var result strings.Builder
	result.WriteString("Channel Information:\n")
	result.WriteString(fmt.Sprintf("ID: %s\n", channel.Id))
	result.WriteString(fmt.Sprintf("Name: %s\n", channel.Name))
	result.WriteString(fmt.Sprintf("Display Name: %s\n", channel.DisplayName))
	result.WriteString(fmt.Sprintf("Type: %s\n", channel.Type))
	result.WriteString(fmt.Sprintf("Team ID: %s\n", channel.TeamId))

	if channel.Purpose != "" {
		result.WriteString(fmt.Sprintf("Purpose: %s\n", channel.Purpose))
	}
	if channel.Header != "" {
		result.WriteString(fmt.Sprintf("Header: %s\n", channel.Header))
	}

	result.WriteString(fmt.Sprintf("Created: %s\n", time.Unix(channel.CreateAt/1000, 0).Format("2006-01-02 15:04:05")))

	// Get member count
	memberCount, _, err := client.GetChannelStats(ctx, channel.Id, "", false)
	if err == nil {
		result.WriteString(fmt.Sprintf("Member Count: %s\n", strconv.FormatInt(memberCount.MemberCount, 10)))
	}

	return result.String(), nil
}

// toolGetChannelMembers implements the get_channel_members tool
func (p *MattermostToolProvider) toolGetChannelMembers(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args GetChannelMembersArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool get_channel_members: %w", err)
	}

	// Validate required fields
	if args.ChannelID == "" {
		return "channel_id is required", fmt.Errorf("channel_id cannot be empty")
	}

	// Set defaults and validate
	if args.Limit == 0 {
		args.Limit = 50
	}
	if args.Limit > 200 {
		args.Limit = 200
	}
	if args.Page < 0 {
		args.Page = 0
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	// Get channel members
	members, _, err := client.GetChannelMembers(ctx, args.ChannelID, args.Page, args.Limit, "")
	if err != nil {
		return "failed to fetch channel members", fmt.Errorf("error fetching channel members: %w", err)
	}

	if len(members) == 0 {
		return "no members found in this channel", nil
	}

	// Get user details for each member
	var result strings.Builder
	result.WriteString(fmt.Sprintf("Channel Members (page %d, showing %d members):\n\n", args.Page, len(members)))

	for i, member := range members {
		user, _, err := client.GetUser(ctx, member.UserId, "")
		if err != nil {
			p.logger.Warn("failed to get user details for member", mlog.String("user_id", member.UserId), mlog.Err(err))
			result.WriteString(fmt.Sprintf("%d. User ID: %s (details unavailable)\n", i+1, member.UserId))
			continue
		}

		result.WriteString(fmt.Sprintf("%d. **%s**", i+1, user.Username))

		if user.FirstName != "" || user.LastName != "" {
			result.WriteString(fmt.Sprintf(" (%s %s)", user.FirstName, user.LastName))
		}

		result.WriteString(fmt.Sprintf("\n   ID: %s\n", user.Id))

		if user.Email != "" {
			result.WriteString(fmt.Sprintf("   Email: %s\n", user.Email))
		}

		// Add role information
		roles := strings.Split(member.Roles, " ")
		if len(roles) > 0 && roles[0] != "" {
			result.WriteString(fmt.Sprintf("   Roles: %s\n", strings.Join(roles, ", ")))
		}

		result.WriteString("\n")
	}

	return result.String(), nil
}

// toolAddUserToChannel implements the add_user_to_channel tool using the context client
func (p *MattermostToolProvider) toolAddUserToChannel(mcpContext *MCPToolContext, argsGetter llm.ToolArgumentGetter) (string, error) {
	var args AddUserToChannelArgs
	err := argsGetter(&args)
	if err != nil {
		return "invalid parameters to function", fmt.Errorf("failed to get arguments for tool add_user_to_channel: %w", err)
	}

	// Validate required fields
	if args.UserID == "" {
		return "user_id is required", fmt.Errorf("user_id cannot be empty")
	}
	if args.ChannelID == "" {
		return "channel_id is required", fmt.Errorf("channel_id cannot be empty")
	}

	// Get client from context
	if mcpContext.Client == nil {
		return "client not available", fmt.Errorf("client not available in context")
	}
	client := mcpContext.Client
	ctx := context.Background()

	// Add user to channel
	_, _, err = client.AddChannelMember(ctx, args.ChannelID, args.UserID)
	if err != nil {
		return "failed to add user to channel", fmt.Errorf("error adding user to channel: %w", err)
	}

	// Get user and channel info for confirmation
	user, _, userErr := client.GetUser(ctx, args.UserID, "")
	channel, _, channelErr := client.GetChannel(ctx, args.ChannelID, "")

	if userErr != nil || channelErr != nil {
		return fmt.Sprintf("Successfully added user %s to channel %s", args.UserID, args.ChannelID), nil
	}

	return fmt.Sprintf("Successfully added user '%s' to channel '%s'", user.Username, channel.DisplayName), nil
}
