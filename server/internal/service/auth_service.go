package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/jwtutil"
	"proman/server/internal/pkg/password"
	"proman/server/internal/repository"
)

type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtExpire time.Duration
}

type LoginResult struct {
	Token     string
	ExpiresAt time.Time
}

func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpire time.Duration) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpire: jwtExpire,
	}
}

func (s *AuthService) EnsureAdmin(ctx context.Context, username, rawPassword string) error {
	count, err := s.userRepo.Count(ctx)
	if err != nil {
		return apperror.Internal(err)
	}
	if count > 0 {
		return nil
	}

	hashed, err := password.Hash(rawPassword)
	if err != nil {
		return apperror.Internal(err)
	}

	admin := &model.User{
		Username:     username,
		PasswordHash: hashed,
	}

	if err := s.userRepo.Create(ctx, admin); err != nil {
		return apperror.Internal(err)
	}

	return nil
}

func (s *AuthService) Login(ctx context.Context, username, rawPassword string) (*LoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || rawPassword == "" {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusUnauthorized, 40101, "用户名或密码错误")
		}
		return nil, apperror.Internal(err)
	}

	if err := password.Compare(user.PasswordHash, rawPassword); err != nil {
		return nil, apperror.New(http.StatusUnauthorized, 40101, "用户名或密码错误")
	}

	token, expiresAt, err := jwtutil.IssueToken(user.ID, user.Username, s.jwtSecret, s.jwtExpire, time.Now())
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &LoginResult{
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}
