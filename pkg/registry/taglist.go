// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry

import (
	"errors"
	"fmt"
	"sort"

	"github.com/gofiber/fiber/v2"

	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

func (r Registry) handleTagList(c *fiber.Ctx) error {
	bid := c.UserContext().Value(bidCtxKey).(types.BlobID)
	n := c.QueryInt("n", -1)

	tags, err := r.store.Tags(bid.Namespace, bid.Repo)
	if err != nil {
		if errors.As(err, &storage.ErrNotFound{}) {
			return c.SendStatus(fiber.StatusNotFound)
		}
		r.log.Error(err, "failed fetching tags from storage")
		return c.Status(fiber.StatusInternalServerError).
			SendString("failed fetching tags from storage")
	}

	sort.Strings(tags)

	if last := c.Query("last", ""); last != "" {
		i := indexOf(last, tags)
		if i == -1 {
			tags = []string{}
		} else {
			tags = tags[i+1:]
		}
	}

	if n >= 0 {
		if n >= len(tags) {
			n = len(tags)
		}
		tags = tags[0:n]
	}

	return c.JSON(types.TagList{
		Name: fmt.Sprintf("%s/%s", bid.Namespace, bid.Repo),
		Tags: tags,
	})
}

func indexOf(of string, s []string) int {
	for idx, e := range s {
		if e == of {
			return idx
		}
	}
	return -1
}
