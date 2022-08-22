package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nmezhenskyi/rcs/internal/cache"
	"github.com/nmezhenskyi/rcs/internal/grpcsrv"
	"github.com/nmezhenskyi/rcs/internal/httpsrv"
	"github.com/nmezhenskyi/rcs/internal/nativesrv"
	"github.com/rs/zerolog"
)

func main() {
	// Main logger instance.
	var logger = zerolog.New(os.Stderr).With().Timestamp().
		Logger().Output(zerolog.ConsoleWriter{Out: os.Stderr})

	configFile := flag.String("c", "rcs.json", "Configuration file")
	devMode := flag.Bool("d", false, "Enable development mode")
	flag.Parse()

	_, err := os.Stat(*configFile)
	if os.IsNotExist(err) {
		logger.Fatal().Err(err).Msg("Configuration file is missing")
	} else if err != nil {
		logger.Fatal().Err(err).Msg("Failed to access configuration file")
	}
	conf, err := readConfig(*configFile)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to parse configuration file")
	}
	if *devMode {
		conf.Verbosity = "dev"
	}

	switch conf.Verbosity {
	case "prod":
		logger = zerolog.New(os.Stderr).
			With().Timestamp().Logger().Level(zerolog.InfoLevel)
	case "dev":
		logger = zerolog.New(os.Stderr).
			With().Timestamp().Logger().Level(zerolog.DebugLevel).
			Output(zerolog.ConsoleWriter{Out: os.Stderr})
	case "none":
		logger = zerolog.New(os.Stderr).Level(zerolog.Disabled)
	}

	var (
		globalCache  *cache.CacheMap
		nativeServer *nativesrv.Server
		httpServer   *httpsrv.Server
		grpcServer   *grpcsrv.Server

		shutdownSignal = make(chan os.Signal, 1)
	)
	signal.Notify(shutdownSignal, syscall.SIGINT, syscall.SIGTERM)

	logger.Info().Msg("--- RCS Started ---")

	globalCache = cache.NewCacheMap()

	if conf.Native.Activate {
		nativeServer = nativesrv.NewServer(globalCache)
		nativeServer.Logger = logger.With().Str("scope", "native").Logger()
		go func() {
			var err error
			if conf.Native.TLS {
				err = nativeServer.ListenAndServeTLS(
					getLocalAddr(conf.Native.Port, conf.Native.OnLocalhost),
					conf.Native.CertFile, conf.Native.KeyFile)
			} else {
				err = nativeServer.ListenAndServe(getLocalAddr(conf.Native.Port, conf.Native.OnLocalhost))
			}
			if err != nil {
				os.Exit(1)
			}
		}()
	}
	if conf.HTTP.Activate {
		httpServer = httpsrv.NewServer(globalCache)
		httpServer.Logger = logger.With().Str("scope", "http").Logger()
		go func() {
			var err error
			if conf.HTTP.TLS {
				err = httpServer.ListenAndServeTLS(
					getLocalAddr(conf.HTTP.Port, conf.HTTP.OnLocalhost),
					conf.HTTP.CertFile, conf.HTTP.KeyFile)
			} else {
				err = httpServer.ListenAndServe(getLocalAddr(conf.HTTP.Port, conf.HTTP.OnLocalhost))
			}
			if err != nil {
				os.Exit(1)
			}
		}()
	}
	if conf.GRPC.Activate {
		grpcServer = grpcsrv.NewServer(globalCache)
		grpcServer.Logger = logger.With().Str("scope", "grpc").Logger()
		go func() {
			var err error
			if conf.GRPC.TLS {
				err = grpcServer.ListenAndServeTLS(
					getLocalAddr(conf.GRPC.Port, conf.GRPC.OnLocalhost),
					conf.GRPC.CertFile, conf.GRPC.KeyFile)
			} else {
				err = grpcServer.ListenAndServe(getLocalAddr(conf.GRPC.Port, conf.GRPC.OnLocalhost))
			}
			if err != nil {
				os.Exit(1)
			}
		}()
	}

	<-shutdownSignal
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if nativeServer != nil {
		nativeServer.Shutdown(ctx)
	}
	if httpServer != nil {
		httpServer.Shutdown(ctx)
	}
	if grpcServer != nil {
		grpcServer.Shutdown(ctx)
	}

	logger.Info().Msg("--- RCS Stopped ---")
}

func getLocalAddr(port int, localhost bool) string {
	if localhost {
		return fmt.Sprintf("localhost:%d", port)
	}
	return fmt.Sprintf(":%d", port)
}
