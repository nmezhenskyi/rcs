package main

import (
	"flag"
	"os"

	"github.com/nmezhenskyi/rcs/internal/httpsrv"
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

	logger.Info().Msg("--- RCS Started ---")

	httpServer := httpsrv.NewServer(nil)
	httpServer.Logger = logger.With().Str("scope", "http").Logger()

	if err := httpServer.ListenAndServe("localhost:5000"); err != nil {
		logger.Fatal().Err(err).Msg("")
	}
}
