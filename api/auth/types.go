package auth

import (
	"hash"

	"github.com/huskyci-org/huskyCI/api/types"
)

// UserCredsHandler is the User handler used in auth.
type UserCredsHandler interface {
	GetPassFromDB(username string) (string, error)
	GetHashedPass(password string) (string, error)
}

// Pbkdf2Generator is the interface that stores all pbkdf2 functions.
type Pbkdf2Generator interface {
	GetCredsFromDB(username string) (types.User, error)
	DecodeSaltValue(salt string) ([]byte, error)
	GenHashValue(value, salt []byte, iter, keyLen int, h hash.Hash) string
	GenerateSalt() (string, error)
	GetHashName() string
	GetIterations() int
	GetKeyLength() int
}

// Pbkdf2Caller is the pbkdf2 caller struct.
type Pbkdf2Caller struct{}

// MongoBasic is the struct that stores the client handler
type MongoBasic struct {
	ClientHandler UserCredsHandler
}

// ClientPbkdf2 is the struct that stores all information regarding a the pbkdf2 client.
type ClientPbkdf2 struct {
	HashGen      Pbkdf2Generator
	Salt         string
	Iterations   int
	KeyLen       int
	HashFunction string
}
