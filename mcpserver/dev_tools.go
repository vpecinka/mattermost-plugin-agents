// Copyright (c) 2023-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package mcpserver

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// DevToolProvider implements development-specific Mattermost operations for MCP tools
type DevToolProvider struct {
	authProvider AuthenticationProvider
	logger       mlog.LoggerIFace
	serverURL    string
}

// NewDevToolProvider creates a new development tool provider
func NewDevToolProvider(authProvider AuthenticationProvider, logger mlog.LoggerIFace, serverURL string) *DevToolProvider {
	return &DevToolProvider{
		authProvider: authProvider,
		logger:       logger,
		serverURL:    serverURL,
	}
}

// createUser implements the create_user development tool
func (p *DevToolProvider) createUser(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract required arguments
	username, ok := arguments["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username is required and must be a string")
	}

	email, ok := arguments["email"].(string)
	if !ok {
		return nil, fmt.Errorf("email is required and must be a string")
	}

	password, ok := arguments["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password is required and must be a string")
	}

	// Extract optional arguments
	firstName := ""
	if val, exists := arguments["first_name"]; exists {
		if str, ok := val.(string); ok {
			firstName = str
		}
	}

	lastName := ""
	if val, exists := arguments["last_name"]; exists {
		if str, ok := val.(string); ok {
			lastName = str
		}
	}

	nickname := ""
	if val, exists := arguments["nickname"]; exists {
		if str, ok := val.(string); ok {
			nickname = str
		}
	}

	// Create user object
	user := &model.User{
		Username:  username,
		Email:     email,
		Password:  password,
		FirstName: firstName,
		LastName:  lastName,
		Nickname:  nickname,
	}

	// Create the user
	createdUser, _, err := client.CreateUser(context.Background(), user)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating user: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("User created successfully! ID: %s, Username: %s, Email: %s", createdUser.Id, createdUser.Username, createdUser.Email),
		}},
	}, nil
}

// createTeam implements the create_team development tool
func (p *DevToolProvider) createTeam(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract required arguments
	name, ok := arguments["name"].(string)
	if !ok {
		return nil, fmt.Errorf("name is required and must be a string")
	}

	displayName, ok := arguments["display_name"].(string)
	if !ok {
		return nil, fmt.Errorf("display_name is required and must be a string")
	}

	teamType, ok := arguments["type"].(string)
	if !ok {
		return nil, fmt.Errorf("type is required and must be a string")
	}

	// Validate team type
	if teamType != "O" && teamType != "I" {
		return nil, fmt.Errorf("type must be 'O' for open or 'I' for invite only")
	}

	// Extract optional arguments
	description := ""
	if val, exists := arguments["description"]; exists {
		if str, ok := val.(string); ok {
			description = str
		}
	}

	// Create team object
	team := &model.Team{
		Name:        name,
		DisplayName: displayName,
		Type:        teamType,
		Description: description,
	}

	// Create the team
	createdTeam, _, err := client.CreateTeam(context.Background(), team)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating team: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Team created successfully! ID: %s, Name: %s, Display Name: %s", createdTeam.Id, createdTeam.Name, createdTeam.DisplayName),
		}},
	}, nil
}

// addUserToTeam implements the add_user_to_team development tool
func (p *DevToolProvider) addUserToTeam(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	userID, ok := arguments["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required and must be a string")
	}

	teamID, ok := arguments["team_id"].(string)
	if !ok {
		return nil, fmt.Errorf("team_id is required and must be a string")
	}

	// Add user to team
	_, _, err := client.AddTeamMember(context.Background(), teamID, userID)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error adding user to team: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("User %s successfully added to team %s", userID, teamID),
		}},
	}, nil
}

// addUserToChannel implements the add_user_to_channel development tool
func (p *DevToolProvider) addUserToChannel(ctx context.Context, client *model.Client4, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract arguments
	userID, ok := arguments["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("user_id is required and must be a string")
	}

	channelID, ok := arguments["channel_id"].(string)
	if !ok {
		return nil, fmt.Errorf("channel_id is required and must be a string")
	}

	// Add user to channel
	_, _, err := client.AddChannelMember(context.Background(), channelID, userID)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error adding user to channel: %v", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("User %s successfully added to channel %s", userID, channelID),
		}},
	}, nil
}

// createPostAsUser implements the create_post_as_user development tool
func (p *DevToolProvider) createPostAsUser(ctx context.Context, arguments map[string]interface{}) (*ToolResult, error) {
	// Extract required arguments
	username, ok := arguments["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username is required and must be a string")
	}

	password, ok := arguments["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password is required and must be a string")
	}

	channelID, ok := arguments["channel_id"].(string)
	if !ok {
		return nil, fmt.Errorf("channel_id is required and must be a string")
	}

	message, ok := arguments["message"].(string)
	if !ok {
		return nil, fmt.Errorf("message is required and must be a string")
	}

	// Extract optional arguments
	rootID := ""
	if val, exists := arguments["root_id"]; exists {
		if str, ok := val.(string); ok {
			rootID = str
		}
	}

	// Extract props (optional)
	var props map[string]interface{}
	if val, exists := arguments["props"]; exists {
		if propsMap, ok := val.(map[string]interface{}); ok {
			props = propsMap
		}
	}

	// Create a new client and login
	client := model.NewAPIv4Client(p.serverURL)

	// Login with username/password
	_, _, err := client.Login(ctx, username, password)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error logging in as user %s: %v. Check that username and password are correct.", username, err),
			}},
			IsError: true,
		}, nil
	}
	// Create the post
	post := &model.Post{
		ChannelId: channelID,
		Message:   message,
		RootId:    rootID,
		Props:     props,
	}

	createdPost, _, err := client.CreatePost(context.Background(), post)
	if err != nil {
		return &ToolResult{
			Content: []Content{{
				Type: "text",
				Text: fmt.Sprintf("Error creating post as user: %v. Check that the user has permission to post in this channel and that the channel_id is correct.", err),
			}},
			IsError: true,
		}, nil
	}

	return &ToolResult{
		Content: []Content{{
			Type: "text",
			Text: fmt.Sprintf("Post created successfully as user! Post ID: %s, Channel: %s", createdPost.Id, createdPost.ChannelId),
		}},
	}, nil
}
