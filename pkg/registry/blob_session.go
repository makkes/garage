// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/makkes/garage/pkg/features"
	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

func (r Registry) handleBlobSessionPost(c *fiber.Ctx) error {
	if c.Request().Header.ContentLength() != 0 {
		r.log.V(8).Info("POST request with non-zero content length", "content-length", c.Request().Header.ContentLength())
	}

	sid, err := r.store.StartSession()
	if err != nil {
		r.log.Error(err, "failed starting session")
		return c.Status(http.StatusInternalServerError).
			SendString("failed starting session")
	}

	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)
	sids := sid.String()

	c.Location(fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", bid.Namespace, bid.Repo, sids))
	return c.SendStatus(fiber.StatusAccepted)
}

func (r Registry) handleBlobGet(c *fiber.Ctx) error {
	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)
	sid, err := uuid.Parse(c.Params("uuid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).
			SendString(fmt.Sprintf("invalid session ID %q", c.Params("uuid")))
	}

	size, err := r.store.GetSessionInfo(sid)
	if err != nil {
		if errors.As(err, &storage.ErrSessionNotFound{}) {
			return c.Status(fiber.StatusNotFound).
				SendString("session not found")
		}
		r.log.Error(err, "failed retrieving session data", "session", sid)
		return c.Status(http.StatusInternalServerError).
			SendString("failed retrieving session data")
	}

	c.Location(fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", bid.Namespace, bid.Repo, sid.String()))
	c.Response().Header.Add("Range", fmt.Sprintf("0-%d", size-1))

	return c.SendStatus(fiber.StatusNoContent)
}

func (r Registry) handleBlobPatch(c *fiber.Ctx) error {
	ct := c.Request().Header.ContentType()
	if len(ct) != 0 && string(ct) != "application/octet-stream" {
		r.log.V(8).Info("POST request with invalid content type", "content-type", ct)
		return c.Status(fiber.StatusBadRequest).
			SendString(fmt.Sprintf("content-type must be 'application/octet-stream' but is %q", string(ct)))
	}

	b := c.Request().BodyStream()
	if b == nil {
		return c.Status(fiber.StatusBadRequest).
			SendString("no data in request body")
	}

	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)
	sid, err := uuid.Parse(c.Params("uuid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).
			SendString(fmt.Sprintf("invalid session ID %q", c.Params("uuid")))
	}

	eor, err := r.store.StoreSessionData(sid, b, c.Get(fiber.HeaderContentRange))
	if err != nil {
		if errors.As(err, &storage.ErrSessionNotFound{}) {
			return c.Status(fiber.StatusNotFound).
				SendString("session not found")
		} else if errors.As(err, &storage.ErrOutOfOrderChunk{}) {
			r.log.Error(err, "out-of-order chunk received")
			return c.SendStatus(fiber.StatusRequestedRangeNotSatisfiable)
		}
		r.log.Error(err, "failed storing session data", "session", sid)
		return c.Status(http.StatusInternalServerError).
			SendString("failed storing session data")
	}

	c.Location(fmt.Sprintf("/v2/%s/%s/blobs/uploads/%s", bid.Namespace, bid.Repo, sid.String()))
	c.Response().Header.Add("Range", fmt.Sprintf("0-%d", eor))

	return c.SendStatus(fiber.StatusAccepted)
}

func (r Registry) handleBlobPut(c *fiber.Ctx) error {
	dig := c.Queries()["digest"]
	if dig == "" {
		return c.Status(fiber.StatusBadRequest).
			SendString("'digest' query parameter missing")
	}

	sid, err := uuid.Parse(c.Params("uuid"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).
			SendString(fmt.Sprintf("invalid session ID %q", c.Params("uuid")))
	}

	b := c.Request().BodyStream()
	if b != nil {
		_, err := r.store.StoreSessionData(sid, b, c.Get(fiber.HeaderContentRange))
		if err != nil {
			if errors.As(err, &storage.ErrSessionNotFound{}) {
				return c.Status(fiber.StatusNotFound).
					SendString("session not found")
			}
			r.log.Error(err, "failed storing session data", "session", sid)
			return c.Status(http.StatusInternalServerError).
				SendString("failed storing session data")
		}
	}

	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)
	resDig, err := r.store.CloseSession(sid, bid)
	if err != nil {
		if errors.As(err, &storage.ErrSessionNotFound{}) {
			return c.Status(fiber.StatusNotFound).
				SendString("session not found")
		}
		r.log.Error(err, "failed closing session", "session", sid)
		return c.Status(http.StatusInternalServerError).
			SendString("failed closing session")
	}

	c.Set(fiber.HeaderLocation, fmt.Sprintf("/v2/%s/%s/blobs/%s", bid.Namespace, bid.Repo, resDig))
	if r.features.Enabled(features.SendLegacyDigestHeader) {
		c.Set("Docker-Content-Digest", resDig.String())
	}

	return c.SendStatus(fiber.StatusCreated)
}
