package handler

import (
	"haskinproxy/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
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

	// 1. Verify captcha
	if !h.authService.VerifyCaptcha(req.CaptchaID, req.CaptchaAnswer) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid captcha"})
		return
	}

	// 2. Send reset code (logic handles rate limiting and privacy)
	err := h.authService.SendResetCode(req.Email)
	if err != nil {
		// If it's a rate limit error, return it
		c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
		return
	}

	// 3. Always return success as per requirement
	c.JSON(http.StatusOK, gin.H{"message": "verification code sent if the email is registered"})
}
