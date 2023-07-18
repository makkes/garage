package main

import (
	"fmt"
	"os"

	"github.com/go-logr/zapr"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	cfgp "github.com/makkes/garage/pkg/cfg"
	"github.com/makkes/garage/pkg/registry"
	"github.com/makkes/garage/pkg/storage"
)

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
	verbosity := cfg.V.GetInt(cfgp.KeyVerbosity)

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
		registry.WithFileStorage(s),
		registry.WithMiddleware(logger.New()),
		registry.WithLogger(log.WithName("registry")),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed creating registry: %s\n", err)
		os.Exit(1)
	}

	laddr := fmt.Sprintf("%s:%d", cfg.V.GetString(cfgp.KeyListenHost), cfg.V.GetInt(cfgp.KeyListenPort))

	fmt.Fprintf(os.Stderr, "starting server at %s\n", laddr)

	if err := r.Start(laddr); err != nil {
		fmt.Fprintf(os.Stderr, "failed starting server: %s\n", err)
		os.Exit(1)
	}
}
