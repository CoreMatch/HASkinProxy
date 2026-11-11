// Package service holds business-logic services. The password-reset
// (auth) service was revived from the retired pre-10426 mainline
// (archive/main-2026-11-04, commits 4ae8a10 + b4a041e) and adapted to the
// current architecture: the upstream user-existence check now calls the
// HRPAuth core's POST /internal/user-exists endpoint.
package service

import (
	"crypto/rand"
	"fmt"
	"haskinproxy/config"
	"haskinproxy/internal/cache"
	"haskinproxy/internal/hrpauth"
	"log"
	"math/big"
	"net/smtp"
	"strings"

	"github.com/mojocn/base64Captcha"
)

var captchaStore = base64Captcha.DefaultMemStore

type AuthService struct {
	cache    *cache.Cache
	haClient *hrpauth.HAClient
}

func NewAuthService(c *cache.Cache, ha *hrpauth.HAClient) *AuthService {
	return &AuthService{
		cache:    c,
		haClient: ha,
	}
}

// GenerateCaptcha generates a 4-character alphanumeric graphic captcha.
func (s *AuthService) GenerateCaptcha() (id, b64s string, err error) {
	driver := base64Captcha.NewDriverString(80, 240, 5, base64Captcha.OptionShowHollowLine, 4, base64Captcha.TxtAlphabet+base64Captcha.TxtNumbers, nil, nil, nil)
	cp := base64Captcha.NewCaptcha(driver, captchaStore)
	id, b64s, _, err = cp.Generate()
	return
}

// VerifyCaptcha verifies the graphic captcha.
func (s *AuthService) VerifyCaptcha(id, answer string) bool {
	return captchaStore.Verify(id, answer, true)
}

// SendResetCode handles the logic of sending a reset code to the email.
func (s *AuthService) SendResetCode(email string) error {
	// 1. Rate limiting check
	if err := s.checkRateLimit(email); err != nil {
		return err
	}

	// 2. Check if user exists (Privacy: always return success later)
	exists, err := s.haClient.CheckUserExistsByEmail(email)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		// If upstream is down, treat as not exists so no mail is sent on
		// uncertainty.
		exists = false
	}

	if !exists {
		// Log for internal tracking but return success to frontend
		log.Printf("Reset code requested for non-existent email: %s", email)
		return nil
	}

	// 3. Generate and store code
	code := s.generateCode()
	err = s.cache.Set("reset_code:"+email, code, config.AppConfig.Auth.CodeTTL)
	if err != nil {
		return fmt.Errorf("failed to store verification code: %v", err)
	}

	// 4. Send email
	go func() {
		err := s.sendEmail(email, code)
		if err != nil {
			log.Printf("Failed to send email to %s: %v", email, err)
		}
	}()

	// 5. Update rate limit counters
	s.updateRateLimit(email)

	return nil
}

func (s *AuthService) checkRateLimit(email string) error {
	// Minute limit
	minKey := "rate:min:" + email
	_, found, _ := s.cache.GetRaw(minKey)
	if found {
		return fmt.Errorf("please wait a minute before requesting another code")
	}

	// Hour limit
	hourKey := "rate:hour:" + email
	var count int
	found, _ = s.cache.Get(hourKey, &count)
	if found && count >= 10 {
		return fmt.Errorf("too many requests, please try again in an hour")
	}

	return nil
}

func (s *AuthService) updateRateLimit(email string) {
	// Minute limit
	s.cache.SetRaw("rate:min:"+email, []byte("1"), 60)

	// Hour limit
	hourKey := "rate:hour:" + email
	var count int
	found, _ := s.cache.Get(hourKey, &count)
	if !found {
		count = 0
	}
	s.cache.Set(hourKey, count+1, 3600)
}

// VerifyResetCode verifies the reset code sent to the email.
func (s *AuthService) VerifyResetCode(email, code string) bool {
	var storedCode string
	found, _ := s.cache.Get("reset_code:"+email, &storedCode)
	if !found {
		return false
	}
	return storedCode == code
}

func (s *AuthService) generateCode() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return fmt.Sprintf("%04d", n.Int64())
}

func (s *AuthService) sendEmail(to, code string) error {
	smtpConfig := config.AppConfig.SMTP
	subject := "Password Reset Verification Code"
	message := fmt.Sprintf("Your verification code is: %s. It will expire in 10 minutes.", code)

	from := smtpConfig.FromEmail
	fromName := smtpConfig.FromName

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/plain; charset=UTF-8"

	var headerStr strings.Builder
	for k, v := range headers {
		headerStr.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	fullMessage := headerStr.String() + "\r\n" + message
	addr := fmt.Sprintf("%s:%d", smtpConfig.Host, smtpConfig.Port)

	var auth smtp.Auth
	if smtpConfig.Username != "" && smtpConfig.Password != "" {
		auth = smtp.PlainAuth("", smtpConfig.Username, smtpConfig.Password, smtpConfig.Host)
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(fullMessage))
}
