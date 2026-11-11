package handler

import (
	"haskinproxy/internal/service"
	"net/http"
	"regexp"

	"github.com/gin-gonic/gin"
)

var (
	captchaRegex   = regexp.MustCompile(`^[a-zA-Z0-9]{4}$`)
	emailCodeRegex = regexp.MustCompile(`^[0-9]{4}$`)
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(as *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: as,
	}
}

type SendResetCodeRequest struct {
	Email         string `json:"email" binding:"required,email"`
	CaptchaID     string `json:"captcha_id" binding:"required"`
	CaptchaAnswer string `json:"captcha_answer" binding:"required"`
}

type VerifyResetCodeRequest struct {
	Email string `json:"email" binding:"required,email"`
	Code  string `json:"code" binding:"required"`
}

// GetCaptcha returns a graphic captcha.
func (h *AuthHandler) GetCaptcha(c *gin.Context) {
	id, b64s, err := h.authService.GenerateCaptcha()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate captcha"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"captcha_id":    id,
		"captcha_image": b64s,
	})
}

// SendResetCode handles the request to send a reset code.
func (h *AuthHandler) SendResetCode(c *gin.Context) {
	var req SendResetCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Verify captcha format
	if !captchaRegex.MatchString(req.CaptchaAnswer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid captcha format, must be 4 alphanumeric characters"})
		return
	}

	// 2. Verify captcha answer
	if !h.authService.VerifyCaptcha(req.CaptchaID, req.CaptchaAnswer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid captcha"})
		return
	}

	// 3. Send reset code (logic handles rate limiting and privacy)
	err := h.authService.SendResetCode(req.Email)
	if err != nil {
		// If it's a rate limit error, return it
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	// 4. Always return success as per requirement
	c.JSON(http.StatusOK, gin.H{"message": "verification code sent if the email is registered"})
}

// VerifyResetCode handles the request to verify the email reset code.
func (h *AuthHandler) VerifyResetCode(c *gin.Context) {
	var req VerifyResetCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. Verify code format
	if !emailCodeRegex.MatchString(req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code format, must be 4 digits"})
		return
	}

	// 2. Verify code
	if !h.authService.VerifyResetCode(req.Email, req.Code) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid or expired verification code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "verification successful"})
}
