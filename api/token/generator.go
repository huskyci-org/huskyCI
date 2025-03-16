package token

import (
	"crypto/rand"
	"encoding/base64"
	"time"

	apiContext "github.com/huskyci-org/huskyCI/api/context"
	"github.com/huskyci-org/huskyCI/api/types"
	"github.com/huskyci-org/huskyCI/api/util"
	"github.com/google/uuid"
)

// ValidateURL validates if an URL is malicious or not.
func (tC *TCaller) ValidateURL(url string) (string, error) {
	return util.CheckMaliciousRepoURL(url)
}

func generateRandomBytes() ([]byte, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return b, err
}

// GenerateToken generates a new token to be used in auth.
func (tC *TCaller) GenerateToken() (string, error) {
	b, err := generateRandomBytes()
	return base64.URLEncoding.EncodeToString(b), err
}

// GetTimeNow returns the time now.
func (tC *TCaller) GetTimeNow() time.Time {
	return time.Now()
}

// StoreAccessToken stores a new access token into MongoDB.
func (tC *TCaller) StoreAccessToken(accessToken types.DBToken) error {
	return apiContext.APIConfiguration.DBInstance.InsertDBAccessToken(accessToken)
}

// FindAccessToken gets an AccessToken based on an given ID.
func (tC *TCaller) FindAccessToken(ID string) (types.DBToken, error) {
	aTokenQuery := map[string]interface{}{"uuid": ID}
	return apiContext.APIConfiguration.DBInstance.FindOneDBAccessToken(aTokenQuery)
}

// FindRepoURL checks if a Access TOken is present based on a given URL.
func (tC *TCaller) FindRepoURL(repositoryURL string) error {
	repoQuery := map[string]interface{}{"repositoryURL": repositoryURL, "isValid": true}
	_, err := apiContext.APIConfiguration.DBInstance.FindOneDBAccessToken(repoQuery)
	return err
}

// GenerateUUID returns a new UUID.
func (tC *TCaller) GenerateUUID() string {
	return uuid.New().String()
}

// EncodeBase64 retunrs a string in base64.
func (tC *TCaller) EncodeBase64(m string) string {
	return base64.URLEncoding.EncodeToString([]byte(m))
}

// DecodeToStringBase64 decodes a base64 string.
func (tC *TCaller) DecodeToStringBase64(encodedVal string) (string, error) {
	decodedVal, err := base64.URLEncoding.DecodeString(encodedVal)
	return string(decodedVal), err
}

// UpdateAccessToken updates an access Token in MongoDB based on its UUID.
func (tC *TCaller) UpdateAccessToken(ID string, accesstoken types.DBToken) error {
	aTokenQuery := map[string]interface{}{"uuid": ID}
	return apiContext.APIConfiguration.DBInstance.UpdateOneDBAccessToken(aTokenQuery, accesstoken)
}
