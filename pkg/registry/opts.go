// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry

import (
	"fmt"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"github.com/makkes/garage/pkg/storage"
)

func (r *Registry) applyDefaults() error {
	if r.store == nil {
		var err error
		r.store, err = storage.NewFileStorage(".", logr.Discard())
		if err != nil {
			return fmt.Errorf("failed instantiating file storage backend: %w", err)
		}
	}

	if r.maxManifestBytes == 0 {
		r.maxManifestBytes = 8 * 1024 * 1024
	}

	if r.log.IsZero() {
		zlog, err := zap.NewDevelopment()
		if err != nil {
			return fmt.Errorf("failed configuring default logger: %w", err)
		}
		r.log = zapr.NewLogger(zlog)
	}

	return nil
}

func WithMaxManifestBytes(b int64) Opt {
	return func(r *Registry) error {
		r.maxManifestBytes = b
		return nil
	}
}

func WithFileStorage(fs storage.Storage) Opt {
	return func(r *Registry) error {
		r.store = fs
		return nil
	}
}

func WithMemStorage() Opt {
	return func(r *Registry) error {
		r.store = storage.NewMemStorage()
		return nil
	}
}

func WithMiddleware(m func(*fiber.Ctx) error) Opt {
	return func(r *Registry) error {
		r.App.Use(m)
		return nil
	}
}

func WithLogger(log logr.Logger) Opt {
	return func(r *Registry) error {
		r.log = log
		return nil
	}
}
