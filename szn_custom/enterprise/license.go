// Custom implementation for Team Edition - allows unlimited AI agents
// Based on mattermost-plugin-agents (Apache 2.0 licensed)
// Modified to enable all features without enterprise license requirements

package enterprise

import (
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// LicenseChecker provides open-source license checking that allows all features.
// This is a custom implementation for the Team Edition that enables unlimited AI agents
// and all features without requiring an enterprise license.
type LicenseChecker struct {
	pluginAPIClient *pluginapi.Client
}

func NewLicenseChecker(pluginAPIClient *pluginapi.Client) *LicenseChecker {
	return &LicenseChecker{
		pluginAPIClient,
	}
}

// IsMultiLLMLicensed always returns true, allowing unlimited AI agents in Team Edition.
func (e *LicenseChecker) IsMultiLLMLicensed() bool {
	return true
}

// IsBasicsLicensed always returns true, allowing all basic AI features in Team Edition.
func (e *LicenseChecker) IsBasicsLicensed() bool {
	return true
}
