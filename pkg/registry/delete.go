// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

func (r Registry) handleManifestDelete(c *fiber.Ctx) error {
	mid := c.UserContext().Value(midCtxKey{}).(types.ManifestID)

	log := r.log.WithValues("namespace", mid.Namespace, "repo", mid.Repo, "tag", mid.Tag, "digest", mid.Digest)

	if err := r.store.DeleteManifest(mid); err != nil {
		if errors.As(err, &storage.ErrNotFound{}) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		log.Error(err, "failed deleting manifest")
		return c.Status(fiber.StatusInternalServerError).
			SendString("failed deleting manifest from storage")
	}

	return c.SendStatus(fiber.StatusAccepted)
}

func (r Registry) handleBlobDelete(c *fiber.Ctx) error {
	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)

	log := r.log.WithValues("namespace", bid.Namespace, "repo", bid.Repo, "digest", bid.Digest)

	if err := r.store.DeleteBlob(bid); err != nil {
		if errors.As(err, &storage.ErrNotFound{}) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		log.Error(err, "failed deleting blob")
		return c.Status(fiber.StatusInternalServerError).
			SendString("failed deleting blob from storage")
	}

	return c.SendStatus(fiber.StatusAccepted)
}
