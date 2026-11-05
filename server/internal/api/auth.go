package api

import (
	"context"

	"gitlab.com/anishgowda21/go-password-manager/server/internal/database"
)

type AuthResponse struct {
	UserID          string     `json:"userId"`
	AccessToken     string     `json:"accessToken"`
	RefreshToken    string     `json:"refreshToken"`
	ExpiresIn       int        `json:"expiresIn"` // seconds
	WrappedVaultKey string     `json:"wrappedVaultKey"`
	Salt            string     `json:"salt"`
	KDF             KDFPayload `json:"kdf"`
}

func (s *server) issueAuth(ctx context.Context, userID, deviceID, deviceName, wrappedKey, salt string, kdf KDFPayload) (AuthResponse, error) {
	refToken, refTokenHash, err := createRefreshToken()
	if err != nil {
		return AuthResponse{}, err
	}

	sessionID, err := database.CreateSession(ctx, s.db, database.SessionRequest{
		UserID:           userID,
		DeviceID:         deviceID,
		DeviceName:       deviceName,
		RefreshTokenHash: refTokenHash,
	})

	if err != nil {
		return AuthResponse{}, err
	}

	accessToken, err := s.newAccessToken(userID, sessionID)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		UserID:          userID,
		AccessToken:     accessToken,
		RefreshToken:    refToken,
		ExpiresIn:       int(accessTokenTTL.Seconds()),
		WrappedVaultKey: wrappedKey,
		Salt:            salt,
		KDF:             kdf,
	}, nil
}
