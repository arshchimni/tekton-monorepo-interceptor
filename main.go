package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/arshchimni/tekton-monorepo-interceptor/diff"
	"github.com/arshchimni/tekton-monorepo-interceptor/log"
	"github.com/arshchimni/tekton-monorepo-interceptor/server"
	"go.uber.org/zap"
)

func main() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	listenerPort := os.Getenv("PORT")
	if listenerPort == "" {
		listenerPort = "9090"
	}
	ctx, cancel := context.WithCancel(context.Background())
	logger, err := log.New(logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create logger: %s\n", err)
		os.Exit(1)
	}
	differ, err := diff.NewDiff(logger)
	if err != nil {
		logger.Fatal("cannot create github client",
			zap.Error(err),
		)
	}

	interceptorServer := server.New(logger, differ)
	interceptorAddr := fmt.Sprintf(":%s", listenerPort)
	listener, err := net.Listen("tcp", interceptorAddr)
	if err != nil {
		logger.Fatal("[ERROR] Failed to listen HTTP port: %s\n", zap.Error(err))
	}
	go func() {
		err = interceptorServer.Serve(listener)
		if err != nil {
			logger.Fatal("[ERROR] Failed to start server: %s\n", zap.Error(err))
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt)

	select {
	case <-sigCh:
		logger.Info("received SIGTERM, exiting gracefully")
	case <-ctx.Done():
	}

	// Gracefully shutdown HTTP server.
	if err := interceptorServer.GracefulStop(ctx); err != nil {
		logger.Fatal("failed to gracefully shutdown monitoring HTTP server", zap.Error(err))
	}
	cancel()

}
