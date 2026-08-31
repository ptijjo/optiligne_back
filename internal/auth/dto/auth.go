package dto

// LoginRequest identifie l'exploitant.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// RefreshRequest porte le refresh token (BFF Next).
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// LogoutRequest révoque un refresh token.
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// User est le compte admin exposé (sans hash).
type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	OperatorCode string `json:"operatorCode"`
	DepotCode    string `json:"depotCode"`
}

// TokenPair est renvoyé au BFF (le refresh ne part pas au navigateur).
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    int    `json:"expiresIn"`
	User         User   `json:"user"`
}
