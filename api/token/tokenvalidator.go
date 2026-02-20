package token

import (
	"github.com/huskyci-org/huskyCI/api/util"
)

// HasAuthorization will verify if exists a valid
// access token for the given repository. If exists,
// it will validate the received access token. A true
// bool is returned if it has authorization. If not,
// it will return false.
func (tV TValidator) HasAuthorization(accessToken, repositoryURL string) bool {
	// Local file analysis (file://) does not require token authentication.
	// The URL is file://<RID> from CLI zip uploads; no repository secret is at stake.
	if util.IsFileURL(repositoryURL) {
		return true
	}
	// Temporary: Verify if exists an access token
	// for that repo
	if err := tV.TokenVerifier.VerifyRepo(repositoryURL); err != nil {
		return true
	}
	if err := tV.TokenVerifier.ValidateToken(accessToken, repositoryURL); err != nil {
		return false
	}
	return true
}
