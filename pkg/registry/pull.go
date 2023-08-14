// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

func (r Registry) handleBlobPull(c *fiber.Ctx) error {
	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)
	log := r.log.WithValues("namespace", bid.Namespace, "repo", bid.Repo, "digest", bid.Digest.String())

	blobRdr, bs, err := r.store.FetchBlob(bid)

	if err != nil {
		if errors.As(err, &storage.ErrNotFound{}) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		log.V(5).Error(err, "failed fetching blob from store")
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	c.Response().Header.Add("Content-Type", "application/octet-stream")
	return c.SendStream(blobRdr, int(bs.Size))
}

func (r Registry) handleManifestPull(c *fiber.Ctx) error {
	mid := c.UserContext().Value(midCtxKey{}).(types.ManifestID)
	log := r.log.WithValues("namespace", mid.Namespace, "repo", mid.Repo, "tag", mid.Tag, "digest", mid.Digest)

	has, err := r.store.Has(mid)
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	if !has {
		log.V(5).Info("request for unknown manifest")
		return fiber.ErrNotFound
	}

	mfRdr, err := r.store.FetchManifest(mid)
	if err != nil {
		log.V(4).Error(err, "failed fetching manifest")
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	var mf map[string]interface{}
	rawMf, err := io.ReadAll(mfRdr)
	if err != nil {
		r.log.Error(err, "failed reading manifest from storage")
		return c.Status(http.StatusInternalServerError).
			SendString("failed reading manifest from storage")
	}

	if err := json.Unmarshal(rawMf, &mf); err != nil {
		r.log.Error(err, "failed decoding manifest to JSON object")
		return c.Status(http.StatusInternalServerError).
			SendString("failed decoding manifest to JSON object")
	}

	mt := "application/vnd.oci.image.manifest.v1+json" // this is the default content type for manifests.
	mtIf, ok := mf["mediaType"]
	if ok {
		if mtFromMf, ok := mtIf.(string); ok {
			mt = mtFromMf
		}
	}

	if c.Accepts(mt) == "" {
		return c.SendStatus(fiber.StatusUnsupportedMediaType)
	}

	c.Context().SetContentType(mt)

	if c.Method() == fiber.MethodHead {
		return c.SendStatus(fiber.StatusOK)
	}

	return c.Send(rawMf)
}
