package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCheckPasswordHash(t *testing.T) {
	password1 := "correctPassword123"
	password2 := "anotherPassword456"
	hash1, _ := HashPassword(password1)
	hash2, _ := HashPassword(password2)

	tests := []struct {
		name          string
		password      string
		hash          string
		wantErr       bool
		matchPassword bool
	}{
		{
			name:          "Correct password",
			password:      password1,
			hash:          hash1,
			wantErr:       false,
			matchPassword: true,
		},
		{
			name:          "Incorrect password",
			password:      "wrongPassword",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Password doesn't match different hash",
			password:      password1,
			hash:          hash2,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Empty password",
			password:      "",
			hash:          hash1,
			wantErr:       false,
			matchPassword: false,
		},
		{
			name:          "Invalid hash",
			password:      password1,
			hash:          "invalidhash",
			wantErr:       true,
			matchPassword: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := CheckPasswordHash(tt.password, tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckPasswordHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && match != tt.matchPassword {
				t.Errorf("CheckPasswordHash() expects %v, got %v", tt.matchPassword, match)
			}
		})
	}
}

func TestMakeAndValidateJWT(t *testing.T) {
	userID1 := "550e8400-e29b-41d4-a716-446655440000"
	userID2 := "660e8400-e29b-41d4-a716-446655440000"
	tokenSecretOK := "supersecretkey"
	tokenSecretBad := "wrongsecretkey"

	tests := []struct {
		Name                string
		userID              uuid.UUID
		expectedUserID      uuid.UUID
		tokenSecret         string
		expectedTokenSecret string
		expiresIn           time.Duration
		wantErr             bool
		matchUserID         bool
	}{
		{
			Name:                "Valid token",
			userID:              uuid.MustParse(userID1),
			expectedUserID:      uuid.MustParse(userID1),
			tokenSecret:         tokenSecretOK,
			expectedTokenSecret: tokenSecretOK,
			expiresIn:           1 * time.Hour,
			wantErr:             false,
			matchUserID:         true,
		},

		{
			Name:                "Invalid token secret",
			userID:              uuid.MustParse(userID1),
			expectedUserID:      uuid.MustParse(userID1),
			tokenSecret:         tokenSecretOK,
			expectedTokenSecret: tokenSecretBad,
			expiresIn:           1 * time.Hour,
			wantErr:             true,
			matchUserID:         true,
		},
		{
			Name:                "Different user ID",
			userID:              uuid.MustParse(userID2),
			expectedUserID:      uuid.MustParse(userID1),
			tokenSecret:         tokenSecretOK,
			expectedTokenSecret: tokenSecretOK,
			expiresIn:           1 * time.Hour,
			wantErr:             false,
			matchUserID:         false,
		},
		{
			Name:                "Expired token",
			userID:              uuid.MustParse(userID1),
			expectedUserID:      uuid.MustParse(userID1),
			tokenSecret:         tokenSecretOK,
			expectedTokenSecret: tokenSecretOK,
			expiresIn:           -1 * time.Hour,
			wantErr:             true,
			matchUserID:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			tokenString, err := MakeJWT(tt.userID, tt.tokenSecret, tt.expiresIn)
			if err != nil {
				t.Fatalf("MakeJWT() error = %v", err)
			}

			returnedUserID, err := ValidateJWT(tokenString, tt.expectedTokenSecret)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJWT() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				match := returnedUserID == tt.expectedUserID
				if match != tt.matchUserID {
					t.Errorf("ValidateJWT() expects userID match %v, got %v", tt.matchUserID, match)
				}
			}
		})
	}
}
