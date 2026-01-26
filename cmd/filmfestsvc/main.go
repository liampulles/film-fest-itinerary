package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Setup zerolog
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("application failed")
	}
}

func run() error {
	log.Info().Msg("Hi!")
	return nil
}
