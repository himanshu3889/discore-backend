package coreApi

import (
	"discore/internal/modules/core/middlewares"
	accountStore "discore/internal/modules/core/store/account"

	"discore/internal/modules/core/models"
	"discore/internal/modules/core/services/authetication/jwtAuthentication"

	"discore/internal/base/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func registerAuthRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	authRoutes(auth)
}

func authRoutes(rg *gin.RouterGroup) {
	rg.POST("signup", SignUp)
	rg.POST("signin", SignIn)
	rg.POST("refresh", GetAccessTokenFromRefresh)
	rg.POST("signout", middlewares.JwtAuthMiddleware(), Signout)
	rg.POST("clerk-signin-signup", middlewares.ClerkRequestMiddleware(), ClerkSigninSignup)
}

// Set tokens in httpOnly cookies
func setRefreshTokenCookies(c *gin.Context, accessToken string, accessTokenValidity int, refreshToken string, refreshTokenValidity int) {
	// Access Token Cookie (short-lived)
	c.SetSameSite(http.SameSiteStrictMode) // CSRF protection
	c.SetCookie(
		"accessToken",
		accessToken,
		accessTokenValidity,
		"",
		"",
		false, // secure (true in production)
		false, // httpOnly
	)
	// Refresh Token Cookie (long-lived)
	if refreshTokenValidity > 0 {
		c.SetCookie(
			"refreshToken",
			refreshToken,
			refreshTokenValidity,
			"/api/auth/refresh",
			"",
			false, // secure (true in production)
			false, // httpOnly
		)
	}
}

// Clear cookies on logout
func clearTokenCookies(c *gin.Context) {
	c.SetCookie("accessToken", "", -1, "/", "", true, true)
	c.SetCookie("refreshToken", "", -1, "/api/auth/refresh", "", true, true)
}

// Build session from request metadata, not JSON
// Note: Without userId
func buildSessionFromMetadata(c *gin.Context, refreshToken string) *models.UserSession {
	rawDeviceInfo := utils.ExtractDeviceInfo(c.Request.UserAgent())
	session := &models.UserSession{
		RefreshToken: refreshToken,
		DeviceInfo:   rawDeviceInfo,
		IPAddress:    c.ClientIP(),
		ExpiresAt:    time.Now().Add(jwtAuthentication.RefreshTokenValidity),
	}
	return session
}

// Generate the user session with access and refresh tokens set in headers
func genererateUserSession(ctx *gin.Context, user *models.User) (sess *models.UserSession, access string, refresh string, err error) {
	// Generate Access Token
	refreshToken, err := jwtAuthentication.GenerateToken(user.Email, user.ID, jwtAuthentication.RefreshTokenValidity, jwtAuthentication.RefreshToken)
	if err != nil {
		return nil, "", "", err
	}

	accessToken, err := jwtAuthentication.GenerateToken(user.Email, user.ID, jwtAuthentication.AccessTokenValidity, jwtAuthentication.AccessToken)
	if err != nil {
		return nil, "", "", err
	}

	session := buildSessionFromMetadata(ctx, refreshToken)
	session.UserID = user.ID
	err = accountStore.CreateSession(ctx, session)
	if err != nil {
		return nil, "", "", err
	}

	// Set new cookies
	setRefreshTokenCookies(ctx, accessToken, int(jwtAuthentication.AccessTokenValidity), refreshToken, int(jwtAuthentication.RefreshTokenValidity.Seconds()))
	return session, accessToken, refreshToken, nil
}

// Create the user helper function
func createUserHelper(ctx *gin.Context, incomingUser *models.User) {
	// TODO: Type checking on the inputs

	// Validate other fields while hashing runs
	if incomingUser.Email == "" || incomingUser.Name == "" {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Missing required fields")
		return
	}

	// Hash Password
	hashedPassword, err := jwtAuthentication.HashPassword(incomingUser.Password)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	var createdUser = &models.User{
		Email:    incomingUser.Email,
		Password: hashedPassword,
		Name:     incomingUser.Name,
		ImageUrl: incomingUser.ImageUrl,
	}

	// Create account
	err = accountStore.CreateUser(ctx, createdUser)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate the user session with tokens
	_, accessToken, refreshToken, err := genererateUserSession(ctx, createdUser)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Send success response with token
	utils.RespondWithSuccess(ctx, http.StatusCreated, gin.H{
		"message":      "Signup Successful",
		"user":         createdUser,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

// User sign-up
// NOTE: Directly generate session but in reality verify the email using the link or code method
func SignUp(ctx *gin.Context) {
	var incomingUser models.User

	// Bind the json with the user struct
	if err := ctx.ShouldBindJSON(&incomingUser); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}
	createUserHelper(ctx, &incomingUser)
}

//

// User Sign-in
func SignIn(ctx *gin.Context) {
	var incomingUser models.User
	// Bind incoming JSON to input struct
	if err := ctx.ShouldBindJSON(&incomingUser); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Find user in DB by email
	user, err := accountStore.GetUserByEmail(ctx, incomingUser.Email)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusNotFound, "User not found")
		return
	}

	// Check password by comparing input password to stored hashed password
	if !jwtAuthentication.CheckPasswordHash(incomingUser.Password, user.Password) {
		utils.RespondWithError(ctx, http.StatusBadRequest, "Invalid password")
		return
	}

	_, accessToken, refreshToken, err := genererateUserSession(ctx, user)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Send success response with token
	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":      "Sign-in Successful",
		"user_id":      user.ID,
		"user_email":   user.Email,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

// User Sign-out
func Signout(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refreshToken")
	if err != nil {
		// Even if no cookie, clear any existing tokens
		clearTokenCookies(c)
		utils.RespondWithSuccess(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
		return
	}

	// Parse token to get email (without validation since we're logging out)
	token, _ := jwt.ParseWithClaims(refreshToken, &jwtAuthentication.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtAuthentication.JwtSecret, nil
	})

	claims, claimsOk := token.Claims.(*jwtAuthentication.JwtClaims)

	user, err := accountStore.GetUserByEmail(c, claims.Email)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	if claimsOk && token.Valid {
		// Remove refresh token from database
		err = accountStore.DeleteUserSession(c, user.ID, refreshToken)
		if err != nil {
			utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	// Clear cookies
	clearTokenCookies(c)

	utils.RespondWithSuccess(c, http.StatusOK, gin.H{
		"message": "Logout successful",
	})
}

// Get the access token using the refresh Token
func GetAccessTokenFromRefresh(c *gin.Context) {
	// Get refresh token from cookie
	refreshToken, err := c.Cookie("refreshToken")
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Refresh token not found in cookies")
		return
	}

	// Parse and validate refresh token
	jwtRefreshToken, err := jwt.ParseWithClaims(refreshToken, &jwtAuthentication.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtAuthentication.JwtSecret, nil
	})
	if err != nil || !jwtRefreshToken.Valid {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid refresh token")
		return
	}

	refreshTokenClaims := jwtRefreshToken.Claims.(*jwtAuthentication.JwtClaims)

	// Verify token type is jwtAuthentication.RefreshToken
	if refreshTokenClaims.Subject != string(jwtAuthentication.RefreshToken) {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid token type")
		return
	}

	// Check if user exists in database and matches
	user, err := accountStore.GetUserByEmail(c, refreshTokenClaims.Email)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "User not found!")
		return
	}

	// Check refreshToken exist or not
	_, err = accountStore.GetUserSessionByToken(c, user.ID, refreshToken)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		// return
	}

	// Generate NEW access token (and optionally new refresh token - token rotation)
	newAccessToken, err := jwtAuthentication.GenerateToken(user.Email, user.ID, jwtAuthentication.AccessTokenValidity, jwtAuthentication.AccessToken)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, err.Error())
		return
	}

	// Set new cookies
	setRefreshTokenCookies(c, newAccessToken, int(jwtAuthentication.AccessTokenValidity), refreshToken, 0)

	utils.RespondWithSuccess(c, http.StatusOK, gin.H{
		"message":      "Access Token refreshed successfully",
		"accessToken":  newAccessToken,
		"refreshToken": refreshToken,
	})
}

// Clerk signin/signup
func ClerkSigninSignup(ctx *gin.Context) {
	var incomingUser models.User

	// Bind the json with the user struct
	if err := ctx.ShouldBindJSON(&incomingUser); err != nil {
		utils.RespondWithError(ctx, http.StatusBadRequest, err.Error())
		return
	}

	// Check user email exists or not
	user_email := incomingUser.Email
	if len(user_email) == 0 {
		utils.RespondWithError(ctx,
			http.StatusBadRequest,
			"Invalid request, send the email")
		return
	}

	// First check if user exist or not
	user, err := accountStore.GetUserByEmail(ctx, incomingUser.Email)

	// If user not found then create user
	if err != nil {
		// Generate random password and store in context
		randomPassword := utils.GenerateSnowflakeID().String()
		incomingUser.Password = randomPassword

		// Call the create user
		createUserHelper(ctx, &incomingUser)
		return
	}

	// Call the SignIn but we need to skip the password matching and re check user by email
	_, accessToken, refreshToken, err := genererateUserSession(ctx, user)
	if err != nil {
		utils.RespondWithError(ctx, http.StatusInternalServerError, err.Error())
		return
	}

	// Send success response with token
	utils.RespondWithSuccess(ctx, http.StatusOK, gin.H{
		"message":      "Sign-in Successful",
		"user_id":      user.ID,
		"user_email":   user.Email,
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})

}
