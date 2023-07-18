package registry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/gofiber/fiber/v2"

	"github.com/makkes/garage/pkg/types"
)

func (r Registry) handleManifestPush(c *fiber.Ctx) error {
	b := c.Request().BodyStream()
	if b == nil {
		return c.Status(fiber.StatusBadRequest).
			SendString("no data in request body")
	}

	rdr := io.LimitReader(b, r.maxManifestBytes)
	body, err := io.ReadAll(rdr)
	if err != nil {
		r.log.V(5).Error(err, "failed reading body")
		return fiber.ErrInternalServerError
	}

	if len(body) == 0 {
		return c.Status(fiber.StatusBadRequest).
			JSON(ErrorResponse{
				Errors: []Error{{
					Code:    ErrCodeManifestInvalid,
					Message: "manifest is empty",
				}},
			})
	}

	// Check if the body contains unread data and if so, return appropriate status code.
	rem := make([]byte, 1)
	read, err := b.Read(rem)
	if err != nil && err != io.EOF {
		r.log.V(5).Error(err, "failed reading another byte from body")
		return fiber.ErrInternalServerError
	}
	if read >= 1 {
		return fiber.ErrRequestEntityTooLarge
	}

	var manifest map[string]interface{}
	if err := json.Unmarshal(body, &manifest); err != nil {
		r.log.V(5).Error(err, "failed unmarshaling body")
		return c.Status(fiber.StatusBadRequest).
			JSON(ErrorResponse{
				Errors: []Error{{
					Code:    ErrCodeManifestInvalid,
					Message: "failed unmarshaling body",
				}},
			})
	}

	// "mediaType", if it exists, must match Content-Type header.
	mtIf, ok := manifest["mediaType"]
	ct := string(c.Request().Header.ContentType())
	if ok {
		mt, ok := mtIf.(string)
		if !ok || ct != mt {
			return c.Status(fiber.StatusBadRequest).
				SendString("Content-Type doesn't match mediaType")
		}
		ct = mt
	}

	if ct == "" {
		return c.Status(fiber.StatusBadRequest).
			SendString("no content-type set")
	}

	mid := c.UserContext().Value(midCtxKey{}).(types.ManifestID)

	var dig types.Digest
	if mid.Digest != nil {
		dig = *mid.Digest
	} else {
		dig, err = types.NewDigest("sha256", bytes.NewReader(body))
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).
				SendString("failed creating digest")
		}
		mid.Digest = &dig
	}

	if err := r.store.StoreManifest(mid, bytes.NewReader(body)); err != nil {
		return fmt.Errorf("failed storing manifest: %w", err)
	}

	c.Set(fiber.HeaderLocation, fmt.Sprintf("/v2/%s/%s/manifests/%s", mid.Namespace, mid.Repo, mid.Ref()))

	return c.SendStatus(fiber.StatusCreated)
}
