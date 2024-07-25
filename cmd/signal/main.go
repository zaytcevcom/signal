package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"signal/internal/app"
	"signal/internal/logger"
	internalhttp "signal/internal/server/http"
	"syscall"
	"time"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "config", "configs/signal/config.json", "Path to configuration file")
}

func main() {
	flag.Parse()

	config, err := LoadConfig(configFile)
	if err != nil {
		fmt.Println("Error loading config: ", err)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer cancel()

	logg := logger.New(config.Logger.Level, nil)

	server := internalhttp.New(logg, app.New(logg), "", config.Port)

	go func() {
		<-ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		if err := server.Stop(ctx); err != nil {
			logg.Error("failed to stop http server: " + err.Error())
		}
	}()

	logg.Info(fmt.Sprintf("Demo service listening on port: %d", config.Port))

	if err := server.Start(ctx); err != nil {
		logg.Error("failed to start http server: " + err.Error())
		cancel()
		os.Exit(1) //nolint:gocritic
	}
}
