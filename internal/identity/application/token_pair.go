package application

// TokenPair is the token payload returned on login and refresh.
type TokenPair struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // access token TTL in seconds
}
