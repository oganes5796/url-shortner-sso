package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/oganes5796/url-shortner-sso/internal/app"
	"github.com/oganes5796/url-shortner-sso/internal/config"
	"github.com/oganes5796/url-shortner-sso/internal/lib/logger/handlers/slogpretty"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	// TODO: Инициализировать объект конфига
	cfg := config.MustLoad()

	// TODO: Инициализировать логгер
	log := setupLogger(cfg.Env)
	log.Info("Starting SSO service",
		slog.Any("cfg", cfg),
	)

	// TODO: Инициальзировать приложения(app)
	application := app.New(log, cfg.GRPC.Port, cfg.StoragePath, cfg.TokenTTL)
	go application.GRPCSrv.MustRun()

	// TODO: Запустить gRPC-сервер приложения

	// TODO: graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop
	log.Info("Received signal", slog.String("signal", sign.String()))
	application.GRPCSrv.Stop()
	log.Info("Stopping SSO service")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
