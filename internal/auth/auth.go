package auth

import (
	"log"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenIssuer = "chirpy"
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

// MakeJWT generates a JWT token for the given user ID and secret key.
func MakeJWT(userID uuid.UUID, tokenSecret string, expiresIn time.Duration) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		Issuer:    TokenIssuer,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		log.Printf("Error signing JWT token: %v", err)
		return "", err
	}
	return signedToken, nil
}

// ValidateJWT validates the given JWT token string using the secret key and returns the user ID if valid.
func ValidateJWT(tokenString, tokenSecret string) (uuid.UUID, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})
	if err != nil {
		log.Printf("Error parsing JWT token: %v", err)
		return uuid.Nil, err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		issuer := claims.Issuer
		if issuer != TokenIssuer {
			log.Printf("Invalid token issuer: expected %s, got %s", TokenIssuer, issuer)
			return uuid.Nil, jwt.ErrTokenInvalidClaims
		}
		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			log.Printf("Error parsing user ID from JWT claims: %v", err)
			return uuid.Nil, err
		}
		return userID, nil
	} else {
		log.Printf("Invalid JWT token")
		return uuid.Nil, err
	}
}
