// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"fmt"
	"math"
	"os"

	"github.com/go-logr/zapr"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	cfgp "github.com/makkes/garage/pkg/cfg"
	"github.com/makkes/garage/pkg/registry"
	"github.com/makkes/garage/pkg/storage"
)

func toInt8(i int) (int8, error) {
	if i > math.MaxInt8 || i < math.MinInt8 {
		return 0, fmt.Errorf("overflow of %d", i)
	}
	return int8(i), nil
}

func main() {
	cfg, err := cfgp.InitViper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed initializing configuration: %s\n", err)
		os.Exit(1)
	}

	if cfg.V.GetBool(cfgp.KeyHelp) {
		cfg.FS.Usage()
		os.Exit(1)
	}

	fsDir := cfg.V.GetString(cfgp.KeyDataDir)
	var verbosity int8
	verbosity, err = toInt8(cfg.V.GetInt(cfgp.KeyVerbosity))
	if err != nil {
		fmt.Fprintf(os.Stderr, "conversion of verbosity flag failed: %s", err)
		os.Exit(1)
	}

	zlog := zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.Lock(os.Stderr),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= zapcore.Level(-verbosity)
			}),
		),
	)
	log := zapr.NewLogger(zlog)

	s, err := storage.NewFileStorage(fsDir, log.WithName("storage"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed creating storage backend: %s\n", err)
		os.Exit(1)
	}

	r, err := registry.New(
		registry.WithFeatures(cfg.Features),
		registry.WithFileStorage(s),
		registry.WithMiddleware(logger.New()),
		registry.WithLogger(log.WithName("registry")),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed creating registry: %s\n", err)
		os.Exit(1)
	}

	laddr := fmt.Sprintf("%s:%d", cfg.V.GetString(cfgp.KeyListenHost), cfg.V.GetInt(cfgp.KeyListenPort))

	start := func() error {
		fmt.Fprintf(os.Stderr, "starting server at %s, serving from %s\n", laddr, fsDir)
		return r.Start(laddr)
	}

	certFile := cfg.V.GetString(cfgp.KeyTLSCertFile)
	keyFile := cfg.V.GetString(cfgp.KeyTLSKeyFile)
	if certFile != "" && keyFile != "" {
		start = func() error {
			fmt.Fprintf(os.Stderr, "starting TLS server at %s, serving from %s\n", laddr, fsDir)
			return r.StartTLS(laddr, certFile, keyFile)
		}
	}

	if err := start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed starting server: %s\n", err)
		os.Exit(1)
	}
}
