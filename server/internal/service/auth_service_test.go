package service

import (
	"context"
	"strconv"
	"testing"
	"time"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/jwtutil"
	"proman/server/internal/repository"
	"proman/server/internal/testutil"
)

func TestAuthServiceEnsureAdminCreatesOnlyWhenEmpty(t *testing.T) {
	db := testutil.OpenMySQL(t)
	userRepo := repository.NewUserRepository(db)
	authService := NewAuthService(userRepo, "secret", 12*time.Hour)

	if err := authService.EnsureAdmin(context.Background(), "admin", "admin123456"); err != nil {
		t.Fatalf("EnsureAdmin first call failed: %v", err)
	}

	count, err := userRepo.Count(context.Background())
	if err != nil {
		t.Fatalf("count users after first EnsureAdmin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 user after first EnsureAdmin, got %d", count)
	}

	user, err := userRepo.FindByUsername(context.Background(), "admin")
	if err != nil {
		t.Fatalf("find created admin: %v", err)
	}
	if user.Username != "admin" {
		t.Fatalf("expected admin username to be kept, got %q", user.Username)
	}

	if err := authService.EnsureAdmin(context.Background(), "another-admin", "another-password"); err != nil {
		t.Fatalf("EnsureAdmin second call failed: %v", err)
	}

	count, err = userRepo.Count(context.Background())
	if err != nil {
		t.Fatalf("count users after second EnsureAdmin: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected EnsureAdmin to skip when users exist, got %d users", count)
	}

	if _, err := userRepo.FindByUsername(context.Background(), "another-admin"); err == nil {
		t.Fatalf("expected second EnsureAdmin not to create another admin")
	}
}

func TestAuthServiceLoginBranches(t *testing.T) {
	db := testutil.OpenMySQL(t)
	userRepo := repository.NewUserRepository(db)
	authService := NewAuthService(userRepo, "secret", 2*time.Hour)

	createdUser := testutil.CreateUser(t, db, "admin", "admin123456")

	testCases := []struct {
		name        string
		username    string
		password    string
		wantCode    int
		wantSuccess bool
	}{
		{
			name:     "empty username returns bad request",
			username: "   ",
			password: "admin123456",
			wantCode: 40001,
		},
		{
			name:     "unknown username returns unauthorized",
			username: "missing",
			password: "admin123456",
			wantCode: 40101,
		},
		{
			name:     "wrong password returns unauthorized",
			username: "admin",
			password: "wrong-password",
			wantCode: 40101,
		},
		{
			name:        "trimmed username logs in and issues jwt",
			username:    "  admin  ",
			password:    "admin123456",
			wantSuccess: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result, err := authService.Login(context.Background(), testCase.username, testCase.password)

			if !testCase.wantSuccess {
				appErr := assertAppErrorCode(t, err, testCase.wantCode)
				if appErr.HTTPStatus == 0 {
					t.Fatalf("expected non-zero http status for code %d", testCase.wantCode)
				}
				if result != nil {
					t.Fatalf("expected nil result on failed login, got %+v", result)
				}
				return
			}

			if err != nil {
				t.Fatalf("Login returned unexpected error: %v", err)
			}
			if result == nil || result.Token == "" {
				t.Fatalf("expected non-empty login result, got %+v", result)
			}

			claims, parseErr := jwtutil.ParseToken(result.Token, "secret")
			if parseErr != nil {
				t.Fatalf("parse issued token: %v", parseErr)
			}
			if claims.Subject != strconv.FormatUint(createdUser.ID, 10) {
				t.Fatalf("expected token subject to be user id %d, got %q", createdUser.ID, claims.Subject)
			}
			if claims.Username != createdUser.Username {
				t.Fatalf("expected token username %q, got %q", createdUser.Username, claims.Username)
			}
			if !result.ExpiresAt.After(time.Now().UTC()) {
				t.Fatalf("expected token expiration to be in the future, got %v", result.ExpiresAt)
			}
		})
	}
}

func assertAppErrorCode(t *testing.T, err error, wantCode int) *apperror.AppError {
	t.Helper()

	appErr := apperror.From(err)
	if appErr == nil {
		t.Fatalf("expected app error code %d, got nil", wantCode)
	}
	if appErr.Code != wantCode {
		t.Fatalf("expected app error code %d, got %d (%v)", wantCode, appErr.Code, err)
	}
	return appErr
}
