package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/big"
	mathrand "math/rand"
	"strings"
	"time"

	"go-permission-system/internal/config"
	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/auth"
	"go-permission-system/internal/pkg/mailer"
	"go-permission-system/internal/pkg/password"
	"go-permission-system/internal/store"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const (
	emailCodeSceneLogin    = "login"
	emailCodeSceneRegister = "register"
)

type AuthService struct {
	userRepo repositoryAuthReader
	redis    *redis.Client
	cfg      config.AuthConfig
	mailer   mailer.Sender
	appName  string
}

type repositoryAuthReader interface {
	GetByID(ctx context.Context, id uint) (*model.User, error)
	GetByUsername(ctx context.Context, username string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
}

func NewAuthService(
	userRepo repositoryAuthReader,
	redisClient *redis.Client,
	cfg config.AuthConfig,
	emailSender mailer.Sender,
	appName string,
) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		redis:    redisClient,
		cfg:      cfg,
		mailer:   emailSender,
		appName:  appName,
	}
}

func (s *AuthService) SendEmailCode(ctx context.Context, req SendEmailCodeRequest) (*SendEmailCodeResponse, error) {
	email := normalizeEmail(req.Email)
	scene := strings.TrimSpace(req.Scene)

	if err := s.ensureEmailCodeDependencies(); err != nil {
		return nil, err
	}
	if err := s.validateScenePreconditions(ctx, scene, email); err != nil {
		return nil, err
	}
	if err := s.ensureEmailCodeCooldown(ctx, scene, email); err != nil {
		return nil, err
	}

	code, err := generateNumericCode(6)
	if err != nil {
		return nil, err
	}

	codeTTL := s.emailCodeTTL()
	cooldownTTL := s.emailCodeCooldown()
	codeKey := store.EmailCodeKey(scene, email)
	cooldownKey := store.EmailCodeCooldownKey(scene, email)

	if err = s.redis.Set(ctx, codeKey, code, codeTTL).Err(); err != nil {
		return nil, err
	}
	if err = s.redis.Set(ctx, cooldownKey, "1", cooldownTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, codeKey).Err()
		return nil, err
	}

	subject, body := s.buildVerificationEmail(scene, code, codeTTL)
	if err = s.mailer.Send(ctx, email, subject, body); err != nil {
		_ = s.redis.Del(ctx, codeKey, cooldownKey).Err()
		return nil, apperror.Internal("failed to send verification code email")
	}

	return &SendEmailCodeResponse{
		Email:      email,
		Scene:      scene,
		ExpiresIn:  int(codeTTL.Seconds()),
		CooldownIn: int(cooldownTTL.Seconds()),
	}, nil
}

// SendEmailCodeWithIP behaves like SendEmailCode but also enforces a per-IP sending limit.
func (s *AuthService) SendEmailCodeWithIP(ctx context.Context, req SendEmailCodeWithIPRequest) (*SendEmailCodeResponse, error) {
	// enforce per-IP send limit (e.g., 20 sends per minute)
	if s.redis != nil {
		ipKey := store.EmailSendIPKey(req.ClientIP)
		limit := int64(20)
		if n, err := s.redis.Incr(ctx, ipKey).Result(); err == nil {
			if n == 1 {
				s.redis.Expire(ctx, ipKey, time.Minute)
			}
			if n > limit {
				return nil, apperror.Conflict("too many verification requests from this IP, try later")
			}
		}
	}

	// Delegate to existing logic
	return s.SendEmailCode(ctx, SendEmailCodeRequest{Email: req.Email, Scene: req.Scene})
}

func (s *AuthService) SendLoginCodeByUsername(ctx context.Context, req SendLoginCodeByUsernameRequest) (*SendEmailCodeResponse, error) {
	username := strings.TrimSpace(req.Username)
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}

	if user.Status != model.StatusEnabled {
		return nil, apperror.Forbidden("user is disabled")
	}

	// Reuse SendEmailCode logic with the user's email
	if user.Email == nil {
		return nil, apperror.BadRequest("user email is not set")
	}
	return s.SendEmailCode(ctx, SendEmailCodeRequest{
		Email: *user.Email,
		Scene: emailCodeSceneLogin,
	})
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (*LoginResponse, error) {
	username := strings.TrimSpace(req.Username)
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.Unauthorized("invalid username or password")
		}
		return nil, err
	}

	if user.Status != model.StatusEnabled {
		return nil, apperror.Forbidden("user is disabled")
	}
	if err = password.Compare(user.Password, req.Password); err != nil {
		return nil, apperror.Unauthorized("invalid username or password")
	}

	// Validate password login code
	if err = s.validatePasswordLoginCode(ctx, req.CaptchaKey, req.Code); err != nil {
		return nil, err
	}
	s.clearPasswordLoginCode(ctx, req.CaptchaKey)

	return s.issueLoginResponse(ctx, user)
}

func (s *AuthService) EmailLogin(ctx context.Context, req EmailLoginRequest) (*LoginResponse, error) {
	email := normalizeEmail(req.Email)
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}

	if user.Status != model.StatusEnabled {
		return nil, apperror.Forbidden("user is disabled")
	}
	if err = s.validateEmailCode(ctx, emailCodeSceneLogin, email, req.Code); err != nil {
		return nil, err
	}
	s.clearEmailCode(ctx, emailCodeSceneLogin, email)

	return s.issueLoginResponse(ctx, user)
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	email := normalizeEmail(req.Email)
	username := strings.TrimSpace(req.Username)
	nickname := strings.TrimSpace(req.Nickname)

	if err := s.ensureUserDoesNotExist(ctx, username, email); err != nil {
		return nil, err
	}
	if err := s.validateEmailCode(ctx, emailCodeSceneRegister, email, req.Code); err != nil {
		return nil, err
	}

	hashedPassword, err := password.Hash(req.Password)
	if err != nil {
		return nil, err
	}

	user := model.User{
		Username: username,
		Email:    &email,
		Password: hashedPassword,
		Nickname: nickname,
		Status:   model.StatusEnabled,
		Roles:    []model.Role{},
	}
	if err = s.userRepo.Create(ctx, &user); err != nil {
		return nil, err
	}

	s.clearEmailCode(ctx, emailCodeSceneRegister, email)

	return &RegisterResponse{
		Message: "registration success, wait for role assignment before entering protected pages",
		User:    NewUserDetailResponse(user),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, tokenID string) error {
	if s.redis == nil {
		return nil
	}
	return s.redis.Del(ctx, store.AccessTokenKey(tokenID)).Err()
}

func (s *AuthService) Me(ctx context.Context, userID uint) (*UserDetailResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}

	response := NewUserDetailResponse(*user)
	return &response, nil
}

func (s *AuthService) issueLoginResponse(ctx context.Context, user *model.User) (*LoginResponse, error) {
	now := time.Now()
	expiresAt := now.Add(time.Duration(s.cfg.AccessTokenTTLMinutes) * time.Minute)
	tokenID := uuid.NewString()

	token, err := auth.GenerateToken(s.cfg.JWTSecret, auth.Claims{
		UserID:   user.ID,
		Username: user.Username,
		TokenID:  tokenID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	})
	if err != nil {
		return nil, err
	}

	if s.redis != nil {
		if err = s.redis.Set(ctx, store.AccessTokenKey(tokenID), user.ID, time.Until(expiresAt)).Err(); err != nil {
			return nil, err
		}
	}

	return &LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      NewUserDetailResponse(*user),
	}, nil
}

// SendPasswordLoginCode generates and stores a 4-digit verification code image in Redis for password login
func (s *AuthService) SendPasswordLoginCode(ctx context.Context, username string) (*SendPasswordLoginCodeResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, apperror.BadRequest("username is required")
	}

	// Check if user exists
	if _, err := s.userRepo.GetByUsername(ctx, username); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.NotFound("user not found")
		}
		return nil, err
	}

	// Check cooldown
	cooldownKey := store.PasswordLoginCodeCooldownKey(username)
	exists, err := s.redis.Exists(ctx, cooldownKey).Result()
	if err != nil {
		return nil, err
	}
	if exists > 0 {
		ttl, err := s.redis.TTL(ctx, cooldownKey).Result()
		if err == nil && ttl > 0 {
			return &SendPasswordLoginCodeResponse{
				CaptchaKey: "", Image: "", ExpiresIn: int(s.emailCodeTTL().Seconds()),
				CooldownIn: int(ttl.Seconds()),
			}, apperror.Conflict("verification code was just sent, please wait before requesting again")
		}
	}

	// Generate 4-digit code
	code, err := generateNumericCode(4)
	if err != nil {
		return nil, err
	}

	// Generate simple captcha image
	width, height := 200, 80
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill background with white
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}

	// Draw random noise
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 100; i++ {
		x := r.Intn(width)
		y := r.Intn(height)
		img.Set(x, y, color.RGBA{uint8(150 + r.Intn(100)), uint8(150 + r.Intn(100)), uint8(150 + r.Intn(100)), 255})
	}

	// Draw random lines
	for i := 0; i < 5; i++ {
		x1 := r.Intn(width)
		y1 := r.Intn(height)
		x2 := r.Intn(width)
		y2 := r.Intn(height)
		drawLine(img, x1, y1, x2, y2, color.RGBA{uint8(100 + r.Intn(155)), uint8(100 + r.Intn(155)), uint8(100 + r.Intn(155)), 255})
	}

	// Draw code with random positions and colors
	charWidth := 40
	startX := 15
	for i, ch := range code {
		x := startX + i*charWidth
		y := 10 + r.Intn(20)
		textColor := color.RGBA{uint8(r.Intn(100)), uint8(r.Intn(100)), uint8(r.Intn(100)), 255}
		drawChar(img, x, y, string(ch), textColor)
	}

	// Convert to PNG bytes
	var buf bytes.Buffer
	if err = png.Encode(&buf, img); err != nil {
		return nil, apperror.Internal("failed to encode captcha image")
	}

	// Convert to base64
	imageBase64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Generate UUID as key
	captchaKey := uuid.New().String()

	// Store in Redis with TTL
	codeTTL := s.emailCodeTTL()
	cooldownTTL := s.emailCodeCooldown()
	codeKey := store.PasswordLoginCodeKey(captchaKey)

	if err = s.redis.Set(ctx, codeKey, code, codeTTL).Err(); err != nil {
		return nil, err
	}
	if err = s.redis.Set(ctx, cooldownKey, "1", cooldownTTL).Err(); err != nil {
		_ = s.redis.Del(ctx, codeKey).Err()
		return nil, err
	}

	return &SendPasswordLoginCodeResponse{
		CaptchaKey: captchaKey,
		Image:      imageBase64,
		ExpiresIn:  int(codeTTL.Seconds()),
		CooldownIn: int(cooldownTTL.Seconds()),
	}, nil
}

// validatePasswordLoginCode verifies the password login code from Redis using captcha key
func (s *AuthService) validatePasswordLoginCode(ctx context.Context, captchaKey, code string) error {
	captchaKey = strings.TrimSpace(captchaKey)
	codeKey := store.PasswordLoginCodeKey(captchaKey)

	storedCode, err := s.redis.Get(ctx, codeKey).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return apperror.Unauthorized("verification code has expired or not sent")
		}
		return err
	}

	if storedCode != strings.TrimSpace(code) {
		return apperror.Unauthorized("invalid verification code")
	}

	return nil
}

// clearPasswordLoginCode removes the password login code from Redis after successful verification
func (s *AuthService) clearPasswordLoginCode(ctx context.Context, captchaKey string) {
	captchaKey = strings.TrimSpace(captchaKey)
	codeKey := store.PasswordLoginCodeKey(captchaKey)
	_ = s.redis.Del(ctx, codeKey).Err()
}

func (s *AuthService) ensureEmailCodeDependencies() error {
	if s.redis == nil {
		return apperror.Internal("redis is required for email verification")
	}
	if s.mailer == nil || !s.mailer.Enabled() {
		return apperror.Internal("mail channel is not configured")
	}
	return nil
}

func (s *AuthService) validateScenePreconditions(ctx context.Context, scene, email string) error {
	switch scene {
	case emailCodeSceneLogin:
		user, err := s.userRepo.GetByEmail(ctx, email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return apperror.NotFound("user not found")
			}
			return err
		}
		if user.Status != model.StatusEnabled {
			return apperror.Forbidden("user is disabled")
		}
	case emailCodeSceneRegister:
		if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
			return apperror.Conflict("email already registered")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
	default:
		return apperror.BadRequest("invalid verification scene")
	}

	return nil
}

func (s *AuthService) ensureEmailCodeCooldown(ctx context.Context, scene, email string) error {
	ttl, err := s.redis.TTL(ctx, store.EmailCodeCooldownKey(scene, email)).Result()
	if err != nil {
		return err
	}
	if ttl > 0 {
		return apperror.BadRequest(fmt.Sprintf("please wait %d seconds before requesting another code", int(ttl.Seconds())))
	}
	return nil
}

func (s *AuthService) validateEmailCode(ctx context.Context, scene, email, code string) error {
	if s.redis == nil {
		return apperror.Internal("redis is required for email verification")
	}

	storedCode, err := s.redis.Get(ctx, store.EmailCodeKey(scene, email)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return apperror.BadRequest("verification code has expired, please request a new one")
		}
		return err
	}

	if strings.TrimSpace(code) != storedCode {
		return apperror.BadRequest("invalid verification code")
	}

	return nil
}

func (s *AuthService) clearEmailCode(ctx context.Context, scene, email string) {
	if s.redis == nil {
		return
	}
	_ = s.redis.Del(ctx, store.EmailCodeKey(scene, email), store.EmailCodeCooldownKey(scene, email)).Err()
}

func (s *AuthService) ensureUserDoesNotExist(ctx context.Context, username, email string) error {
	if _, err := s.userRepo.GetByEmail(ctx, email); err == nil {
		return apperror.Conflict("email already registered")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return apperror.Conflict("username already exists")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	return nil
}

func (s *AuthService) emailCodeTTL() time.Duration {
	if s.cfg.EmailCodeTTLSeconds <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(s.cfg.EmailCodeTTLSeconds) * time.Second
}

func (s *AuthService) emailCodeCooldown() time.Duration {
	if s.cfg.EmailCodeCooldownSeconds <= 0 {
		return time.Minute
	}
	return time.Duration(s.cfg.EmailCodeCooldownSeconds) * time.Second
}

func (s *AuthService) buildVerificationEmail(scene, code string, ttl time.Duration) (string, string) {
	sceneLabel := "login"
	if scene == emailCodeSceneRegister {
		sceneLabel = "registration"
	}

	subject := fmt.Sprintf("[%s] %s verification code", s.safeAppName(), strings.Title(sceneLabel))
	body := strings.Join([]string{
		fmt.Sprintf("You are using %s for %s.", s.safeAppName(), sceneLabel),
		fmt.Sprintf("Verification code: %s", code),
		fmt.Sprintf("Expires in: %d minutes", int(ttl.Minutes())),
		"If you did not request this code, you can ignore this email.",
	}, "\n")

	return subject, body
}

func (s *AuthService) safeAppName() string {
	if strings.TrimSpace(s.appName) == "" {
		return "YunShu CMDB"
	}
	return strings.TrimSpace(s.appName)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateNumericCode(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("length must be positive")
	}

	max := big.NewInt(1)
	for i := 0; i < length; i++ {
		max.Mul(max, big.NewInt(10))
	}

	number, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%0*d", length, number.Int64()), nil
}

// drawLine draws a line on the image
func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy

	for {
		if x1 >= 0 && x1 < img.Bounds().Dx() && y1 >= 0 && y1 < img.Bounds().Dy() {
			img.Set(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

// drawChar draws a character on the image with larger pixels for better visibility
func drawChar(img *image.RGBA, x, y int, ch string, c color.RGBA) {
	charMap := map[rune][][]bool{
		'0': {
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
		},
		'1': {
			{false, false, true, false, false},
			{false, true, true, false, false},
			{true, true, true, false, false},
			{false, true, true, false, false},
			{false, true, true, false, false},
			{false, true, true, false, false},
			{true, true, true, true, true},
		},
		'2': {
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, true},
		},
		'3': {
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
		},
		'4': {
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
		},
		'5': {
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
		},
		'6': {
			{true, true, true, true, true},
			{true, false, false, false, false},
			{true, false, false, false, false},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
		},
		'7': {
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, true, false},
			{false, false, true, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
			{false, true, false, false, false},
		},
		'8': {
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
		},
		'9': {
			{true, true, true, true, true},
			{true, false, false, false, true},
			{true, false, false, false, true},
			{true, true, true, true, true},
			{false, false, false, false, true},
			{false, false, false, false, true},
			{true, true, true, true, true},
		},
	}

	if bitmap, ok := charMap[rune(ch[0])]; ok {
		pixelSize := 6
		for i, row := range bitmap {
			for j, pixel := range row {
				if pixel {
					for py := 0; py < pixelSize; py++ {
						for px := 0; px < pixelSize; px++ {
							tx := x + j*pixelSize + px
							ty := y + i*pixelSize + py
							if tx >= 0 && tx < img.Bounds().Dx() && ty >= 0 && ty < img.Bounds().Dy() {
								img.Set(tx, ty, c)
							}
						}
					}
				}
			}
		}
	}
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
