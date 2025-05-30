package app

import (
	"log/slog"
	"time"

	grpcapp "github.com/oganes5796/url-shortner-sso/internal/app/grpc"
	"github.com/oganes5796/url-shortner-sso/internal/services/auth"
	"github.com/oganes5796/url-shortner-sso/internal/storage/sqlite"
)

type App struct {
	GRPCSrv *grpcapp.App
}

func New(log *slog.Logger, grpcPort int, storagePath string, tokenTTL time.Duration) *App {
	// TODO: Инициализировать хранилище (storage)
	storage, err := sqlite.New(storagePath)
	if err != nil {
		panic(err)
	}

	// TODO: Инициализировать auth сервис (auth)
	authService := auth.New(log, storage, storage, storage, tokenTTL)

	grpcApp := grpcapp.New(log, authService, grpcPort)

	return &App{
		GRPCSrv: grpcApp,
	}
}
