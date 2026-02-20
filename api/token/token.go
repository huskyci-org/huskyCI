package token

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"

	"github.com/huskyci-org/huskyCI/api/auth"
	"github.com/huskyci-org/huskyCI/api/types"
)

// GenerateAccessToken will generate a valid access token
// for a the requested repository URL. The access token
// consists in two parts. The first is the UUID that is
// used for identification in DB. The second part is a
// random data. The hash of the random data is stored
// using PBKDF2 algorithm. It is returned the base64 of
// the two parts separated by two points.
// If repositoryURL is empty, a generic token will be created
// that can be used with any repository.
func (tH *THandler) GenerateAccessToken(repo types.TokenRequest) (string, error) {
	accessToken := types.DBToken{}
	validatedURL, err := tH.External.ValidateURL(repo.RepositoryURL)
	if err != nil {
		return "", err
	}
	// Empty URL is now valid - it creates a generic token
	// that can be used with any repository
	token, err := tH.External.GenerateToken()
	if err != nil {
		return "", err
	}
	salt, err := tH.HashGen.GenerateSalt()
	if err != nil {
		return "", err
	}
	bSalt, err := tH.HashGen.DecodeSaltValue(salt)
	if err != nil {
		return "", err
	}
	keyLength := tH.HashGen.GetKeyLength()
	iterations := tH.HashGen.GetIterations()
	accessToken.HuskyToken = tH.HashGen.GenHashValue([]byte(token), bSalt, iterations, keyLength, sha256.New())
	accessToken.URL = validatedURL
	accessToken.IsValid = true
	accessToken.CreatedAt = tH.External.GetTimeNow()
	accessToken.Salt = salt
	accessToken.UUID = tH.External.GenerateUUID()
	if err := tH.External.StoreAccessToken(accessToken); err != nil {
		return "", err
	}
	return tH.External.EncodeBase64(fmt.Sprintf("%s:%s", accessToken.UUID, token)), nil
}

// GetSplitted will return UUID and random part
// of the received access token. It will decode
// the base64 first. The first argument returned
// is the UUID and the second is the random data.
func (tH *THandler) GetSplitted(rcvToken string) (string, string, error) {
	decodedToken, err := tH.External.DecodeToStringBase64(rcvToken)
	if err != nil {
		return "", "", err
	}
	parsed := strings.Split(decodedToken, ":")
	if len(parsed) != 2 {
		return "", "", errors.New("Invalid access token format")
	}
	return parsed[0], parsed[1], nil
}

// ValidateRandomData will calculate the hash from the
// received data and compare with hashdata passed in
// the argument. The hash calculated uses the salt
// passed in the argument.
func (tH *THandler) ValidateRandomData(rdata, hashdata, salt string) error {
	bSalt, err := tH.HashGen.DecodeSaltValue(salt)
	if err != nil {
		return err
	}
	hashFunction := tH.HashGen.GetHashName()
	h, isOk := auth.GetValidHashFunction(hashFunction)
	if !isOk {
		return errors.New("Invalid hash function")
	}
	keyLength := tH.HashGen.GetKeyLength()
	iterations := tH.HashGen.GetIterations()
	hashval := tH.HashGen.GenHashValue([]byte(rdata), bSalt, iterations, keyLength, h)
	if hashval != hashdata {
		return errors.New("Hash value from random data is different")
	}
	return nil
}

// ValidateToken will validate the received token.
// It will verify if it exists an entry through the
// returned UUID. If it exists, it will verify if it
// is a valid token. It will verify the access token
// has permission to start an analysis for the received
// repository URL.
// If the token's URL is empty, it's a generic token
// that can be used with any repository.
func (tH *THandler) ValidateToken(token, repositoryURL string) error {
	validURL, err := tH.External.ValidateURL(repositoryURL)
	if err != nil {
		return err
	}
	uUID, randomData, err := tH.GetSplitted(token)
	if err != nil {
		return err
	}
	accessToken, err := tH.External.FindAccessToken(uUID)
	if err != nil {
		return err
	}
	if !accessToken.IsValid {
		return errors.New("Access token is invalid")
	}
	// If token's URL is empty, it's a generic token that works with any repository
	// Otherwise, check for exact match
	if accessToken.URL != "" && accessToken.URL != validURL {
		return errors.New("Access token doesn't have permission to run analysis in the provided repository")
	}
	return tH.ValidateRandomData(randomData, accessToken.HuskyToken, accessToken.Salt)
}

// VerifyRepo will verify if exists an entry
// for the received repository. It also checks for generic tokens
// (tokens with empty URL) that can work with any repository.
func (tH *THandler) VerifyRepo(repositoryURL string) error {
	validURL, err := tH.External.ValidateURL(repositoryURL)
	if err != nil {
		return err
	}
	// First check for repository-specific token
	err = tH.External.FindRepoURL(validURL)
	if err == nil {
		return nil
	}
	// If no repository-specific token found, check for generic tokens
	// Generic tokens have empty repositoryURL
	err = tH.External.FindRepoURL("")
	if err == nil {
		return nil
	}
	// Neither repository-specific nor generic token found
	return err
}

// InvalidateToken will set boolean flag IsValid
// to false if the passed access token is found
// in DB.
func (tH *THandler) InvalidateToken(token string) error {
	uUID, _, err := tH.GetSplitted(token)
	if err != nil {
		return err
	}
	accessToken, err := tH.External.FindAccessToken(uUID)
	if err != nil {
		return err
	}
	accessToken.IsValid = false
	return tH.External.UpdateAccessToken(uUID, accessToken)
}
