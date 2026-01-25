package auth

import (
	"log"

	"github.com/alexedwards/argon2id"
)

// HashPassword hashes the given plain-text password using Argon2id.
func HashPassword(password string) (string, error) {
	// Use default parameters for Argon2id
	params := argon2id.DefaultParams
	hashedPassword, err := argon2id.CreateHash(password, params)
	if err != nil {
		log.Printf("Error hashing password: %v", err)
		return "", err
	}
	return hashedPassword, nil
}

// CheckPasswordHash compares a plain-text password with its hashed version.
func CheckPasswordHash(password, hashedPassword string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(password, hashedPassword)
	if err != nil {
		log.Printf("Error comparing password and hash: %v", err)
		return false, err
	}
	return match, nil
}
