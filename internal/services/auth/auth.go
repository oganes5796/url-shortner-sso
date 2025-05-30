package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/oganes5796/url-shortner-sso/internal/domain/models"
	"github.com/oganes5796/url-shortner-sso/internal/lib/jwt"
	"github.com/oganes5796/url-shortner-sso/internal/lib/logger/sl"
	"github.com/oganes5796/url-shortner-sso/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type UserSaver interface {
	SaveUser(ctx context.Context, email string, passHash []byte) (uid int64, err error)
}

type UserProvider interface {
	User(ctx context.Context, email string) (models.User, error)
	IsAdmin(ctx context.Context, userID int64) (bool, error)
}

type AppProvider interface {
	App(ctx context.Context, appID int) (models.App, error)
}

type Auth struct {
	log          *slog.Logger
	UserSaver    UserSaver
	UserProvider UserProvider
	AppProvider  AppProvider
	tokenTTL     time.Duration
}

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidAppID       = errors.New("invalid app ID")
	ErrUserExists         = errors.New("user already exists")
)

// New returns a new instance of the Auth server
func New(log *slog.Logger, userSaver UserSaver, userProvider UserProvider, appProvider AppProvider, tokenTTL time.Duration) *Auth {
	return &Auth{
		log:          log,
		UserSaver:    userSaver,
		UserProvider: userProvider,
		AppProvider:  appProvider,
		tokenTTL:     tokenTTL,
	}
}

// Login checks if the user with the given email and password exists
//
// If the user exists, but password is incorrect, returns error.
// If the user does not exist, it returns an error.
func (a *Auth) Login(ctx context.Context, email, password string, appID int) (token string, err error) {
	const op = "auth.Login"

	log := a.log.With(slog.String("op", op), slog.String("email", email))

	log.Info("Logging in user")

	user, err := a.UserProvider.User(ctx, email)
	if err != nil {
		if errors.Is(err, storage.ErrorUserNotFound) {
			a.log.Warn("user not found", sl.Err(err))
			return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
		}

		a.log.Error("failed to get user", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	if err := bcrypt.CompareHashAndPassword(user.PassHash, []byte(password)); err != nil {
		a.log.Info("invalid credentials", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, ErrInvalidCredentials)
	}

	app, err := a.AppProvider.App(ctx, appID)
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	log.Info("User logged in successfully")

	token, err = jwt.NewToken(user, app, a.tokenTTL)
	if err != nil {
		log.Error("failed to create token", sl.Err(err))
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return token, nil
}

// RegisterNewUser saves a new user with the given email and password
//
// If the user already exists, it returns an error.
// If the user does not exist, it saves the user and returns the user ID.
func (a *Auth) RegisterNewUser(ctx context.Context, email, password string) (userID int64, err error) {
	const op = "auth.RegisterNewUser"

	log := a.log.With(slog.String("op", op), slog.String("email", email))

	log.Info("Registering new user")

	passHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("failed to generate password hash", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	id, err := a.UserSaver.SaveUser(ctx, email, passHash)
	if err != nil {
		if errors.Is(err, storage.ErrorUserExists) {
			log.Warn("user already exists", sl.Err(err))
			return 0, fmt.Errorf("%s: %w", op, ErrUserExists)
		}
		log.Error("failed to save user", sl.Err(err))
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("User registered successfully")
	return id, nil
}

// IsAdmin checks if the user with the given userID is an admin
//
// If the user is an admin, it returns true.
// If the user is not an admin, it returns false.
// If the user does not exist, it returns an error.
func (a *Auth) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	const op = "auth.IsAdmin"

	log := a.log.With(slog.String("op", op), slog.Int64("userID", userID))
	log.Info("Checking if user is admin")

	isAdmin, err := a.UserProvider.IsAdmin(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrorAppNotFound) {
			log.Warn("user not found", sl.Err(err))
			return false, fmt.Errorf("%s: %w", op, ErrInvalidAppID)
		}
		log.Error("failed to check if user is admin", sl.Err(err))
		return false, fmt.Errorf("%s: %w", op, err)
	}
	log.Info("User admin status checked", slog.Bool("isAdmin", isAdmin))
	return isAdmin, nil
}
