// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package cfg

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/makkes/garage/pkg/features"
)

const (
	KeyListenHost  = "host"
	KeyListenPort  = "port"
	KeyDataDir     = "data-dir"
	KeyVerbosity   = "verbosity"
	KeyHelp        = "help"
	KeyTLSCertFile = "tls-cert-file"
	KeyTLSKeyFile  = "tls-key-file"
)

type Config struct {
	V        *viper.Viper
	FS       *pflag.FlagSet
	Features features.Features
}

func InitViper() (Config, error) {
	cfg := Config{
		V: viper.New(),
	}

	cfg.V.SetDefault(KeyListenHost, "0.0.0.0")
	cfg.V.SetDefault(KeyListenPort, 8080)
	cfg.V.SetDefault(KeyDataDir, "data")

	cfg.V.AddConfigPath(".")
	if err := cfg.V.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return cfg, fmt.Errorf("failed reading config: %w", err)
		}
	}

	cfg.V.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	cfg.V.AutomaticEnv()

	cfg.FS = pflag.NewFlagSet("default", pflag.ContinueOnError)
	cfg.FS.String(KeyListenHost, cfg.V.GetString(KeyListenHost), "Host to bind to")
	cfg.FS.IntP(KeyListenPort, "p", cfg.V.GetInt(KeyListenPort), "Port to bind to")
	cfg.FS.String(KeyDataDir, cfg.V.GetString(KeyDataDir), "Directory for storing all data")
	cfg.FS.IntP(KeyVerbosity, "v", cfg.V.GetInt(KeyVerbosity), "Number for the log level verbosity (higher is more verbose)")
	cfg.FS.String(KeyTLSCertFile, cfg.V.GetString(KeyTLSCertFile), "Certificate file for serving HTTPS")
	cfg.FS.String(KeyTLSKeyFile, cfg.V.GetString(KeyTLSKeyFile), "Key file for serving HTTPS")
	cfg.FS.BoolP(KeyHelp, "h", false, "Show this help")

	cfg.Features = features.Features{}
	cfg.Features.BindFlags(cfg.FS)

	if err := cfg.FS.Parse(os.Args[1:]); err != nil {
		return cfg, fmt.Errorf("failed parsing command-line flags: %w", err)
	}

	if err := cfg.V.BindPFlags(cfg.FS); err != nil {
		return cfg, fmt.Errorf("failed binding flag set: %w", err)
	}

	cfg.FS.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		cfg.FS.PrintDefaults()
	}
	return cfg, nil
}
