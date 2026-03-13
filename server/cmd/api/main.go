package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"proman/server/internal/app"
)

func main() {
	logger := log.New(os.Stdout, "[proman] ", log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC)

	application, err := app.New(logger)
	if err != nil {
		logger.Fatalf("bootstrap app: %v", err)
	}

	runErrCh := make(chan error, 1)
	go func() {
		logger.Printf("server listening on :%s", application.Config.HTTPPort)
		runErrCh <- application.Run()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-runErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatalf("run app: %v", err)
		}
		logger.Println("server stopped")
		return
	case <-ctx.Done():
		logger.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(shutdownCtx); err != nil {
		logger.Fatalf("shutdown app: %v", err)
	}

	if err := <-runErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatalf("server stopped with error: %v", err)
	}

	logger.Println("server stopped")
}
