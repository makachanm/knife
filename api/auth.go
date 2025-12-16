package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"

	"knife/base"
	"knife/db"

	"golang.org/x/crypto/bcrypt"
)

type AuthAPI struct {
	ProfileModel *db.ProfileModel
	SecretKey    []byte // Secret key for HMAC
}

func NewAuthAPI(profileModel *db.ProfileModel, secretKey string) *AuthAPI {
	return &AuthAPI{
		ProfileModel: profileModel,
		SecretKey:    []byte(secretKey),
	}
}

func (a *AuthAPI) RegisterHandlers(router *base.APIRouter) {
	router.POST("login", a.loginHandler, nil)
	router.POST("logout", a.logoutHandler, nil)
	router.GET("auth/status", a.statusHandler, nil)
}

func (a *AuthAPI) loginHandler(ctx base.APIContext) {
	var req struct {
		Password string `json:"password"`
	}
	if err := ctx.GetContext(&req); err != nil {
		ctx.ReturnError("invalid_request", "Invalid request payload", http.StatusBadRequest)
		return
	}

	// Fetch the stored password hash from the profile
	profile, err := a.ProfileModel.Get()
	if err != nil {
		ctx.ReturnError("server_error", "Failed to fetch profile", http.StatusInternalServerError)
		return
	}

	// Verify the password
	if err := bcrypt.CompareHashAndPassword([]byte(profile.PasswordHash), []byte(req.Password)); err != nil {
		fmt.Println("Stored hash:", profile.PasswordHash)
		fmt.Println("Input password:", req.Password)
		fmt.Println("Error:", err)
		ctx.ReturnError("unauthorized", "Invalid password", http.StatusUnauthorized)
		return
	}

	// Generate an HMAC token based on the password hash
	token := a.generateToken(profile.PasswordHash)

	// Set the token in a secure cookie
	ctx.SetCookie("auth_token", token, "/", 24*60*60, true)
	ctx.ReturnJSON(map[string]string{"message": "Login successful"})
}

func (a *AuthAPI) logoutHandler(ctx base.APIContext) {
	// Clear the auth token cookie
	ctx.SetCookie("auth_token", "", "/", -1, true)
	ctx.ReturnJSON(map[string]string{"message": "Logout successful"})
}

func (a *AuthAPI) statusHandler(ctx base.APIContext) {
	cookie, err := ctx.GetCookie("auth_token")
	if err != nil || !a.validateToken(cookie.Value) {
		ctx.ReturnJSON(map[string]bool{"logged_in": false})
		return
	}
	ctx.ReturnJSON(map[string]bool{"logged_in": true})
}

func (a *AuthAPI) generateToken(passwordHash string) string {
	h := hmac.New(sha256.New, a.SecretKey)
	h.Write([]byte(passwordHash))
	return hex.EncodeToString(h.Sum(nil))
}

func (a *AuthAPI) validateToken(token string) bool {
	profile, err := a.ProfileModel.Get()
	if err != nil {
		return false
	}
	expectedToken := a.generateToken(profile.PasswordHash)
	return hmac.Equal([]byte(token), []byte(expectedToken))
}

// AuthMiddleware implements base.APIMiddleware
type AuthMiddleware struct {
	AuthAPI *AuthAPI
}

func NewAuthMiddleware(authAPI *AuthAPI) *AuthMiddleware {
	return &AuthMiddleware{AuthAPI: authAPI}
}

func (m *AuthMiddleware) RunMiddleware(w http.ResponseWriter, r *http.Request) base.APIMiddlewareResult {
	cookie, err := r.Cookie("auth_token")
	if err != nil || !m.AuthAPI.validateToken(cookie.Value) {
		return base.APIMiddlewareResult{
			IsSuccess: false,
			ApiError:  base.NewAPIError("unauthorized", "Unauthorized access", http.StatusUnauthorized),
		}
	}
	return base.APIMiddlewareResult{IsSuccess: true}
}

func (m *AuthMiddleware) GetMiddlewareInfo() base.APIMiddlewareInfo {
	return base.APIMiddlewareInfo{MiddlewareName: "AuthMiddleware"}
}
