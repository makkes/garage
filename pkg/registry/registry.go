package registry

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

const (
	NamespaceRegex = `^[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*(/[a-z0-9]+((\.|_|__|-+)[a-z0-9]+)*)*$`
	TagRegex       = `^[a-zA-Z0-9_][a-zA-Z0-9._-]{0,127}$`
	DigestRegex    = `^[a-z0-9]+([+._-][a-z0-9]+)*:[a-zA-Z0-9=_-]+`
)

type Opt func(r *Registry) error

type midCtxKey struct{}

type ctxKey int

const (
	bidCtxKey ctxKey = iota
)

type Registry struct {
	App              *fiber.App
	log              logr.Logger
	nsRE             *regexp.Regexp
	tagRE            *regexp.Regexp
	digRE            *regexp.Regexp
	maxManifestBytes int64
	store            storage.Storage
	uploadSessions   map[string]string
}

func New(opts ...Opt) (Registry, error) {
	r := Registry{
		App: fiber.New(fiber.Config{
			DisableStartupMessage: true,
		}),
		nsRE:           regexp.MustCompile(NamespaceRegex),
		tagRE:          regexp.MustCompile(TagRegex),
		digRE:          regexp.MustCompile(DigestRegex),
		uploadSessions: make(map[string]string),
	}
	r.App.Server().StreamRequestBody = true
	r.App.Use(recover.New())

	for _, opt := range opts {
		if err := opt(&r); err != nil {
			return r, fmt.Errorf("failed applying option: %w", err)
		}
	}

	if err := r.applyDefaults(); err != nil {
		return r, fmt.Errorf("failed applying default config: %w", err)
	}

	v2 := r.App.Group("/v2")
	v2.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	v2.Get("/+/tags/list", r.validateNamespacePath, r.handleTagList)

	mr := v2.Group("/+/manifests/:ref", r.validateManifestPath)
	mr.Get("", r.handleManifestPull)
	mr.Put("", r.handleManifestPush)
	mr.Delete("", r.handleManifestDelete)

	br := v2.Group("/+/blobs/")
	br.Post("uploads/", r.validateNamespacePath, r.handleBlobSessionPost)
	br.Patch("uploads/:uuid", r.validateNamespacePath, r.handleBlobPatch)
	br.Put("uploads/:uuid", r.validateNamespacePath, r.handleBlobPut)
	br.Get("uploads/:uuid", r.validateNamespacePath, r.handleBlobGet)
	br.Get(":dig", r.validateBlobPath, r.handleBlobPull)
	br.Delete(":dig", r.validateBlobPath, r.handleBlobDelete)

	return r, nil
}

func (r Registry) Start(addr string) error {
	return r.App.Listen(addr)
}

func (r Registry) Test(req *http.Request) (*http.Response, error) {
	return r.App.Test(req)
}

func (r Registry) validateNamespacePath(c *fiber.Ctx) error {
	nameP := c.Params("+1")
	if !r.nsRE.MatchString(nameP) {
		return fiber.NewError(fiber.StatusBadRequest, "wrong path")
	}

	ns, repo, err := parseName(nameP)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("failed parsing name: %s", err))
	}

	c.SetUserContext(context.WithValue(c.UserContext(), bidCtxKey, types.BlobID{
		Namespace: ns,
		Repo:      repo,
	}))

	return c.Next()
}

func (r Registry) validateBlobPath(c *fiber.Ctx) error {
	nameP := c.Params("+1")
	digP := c.Params("dig")
	if !r.nsRE.MatchString(nameP) || !r.digRE.MatchString(digP) {
		return fiber.NewError(fiber.StatusBadRequest, "wrong path")
	}

	var dig types.Digest
	sp := strings.Split(digP, ":")
	dig.Algo = sp[0]
	dig.Enc = sp[1]

	ns, repo, err := parseName(nameP)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("failed parsing name: %s", err))
	}

	c.SetUserContext(context.WithValue(c.UserContext(), bidCtxKey, types.BlobID{
		Namespace: ns,
		Repo:      repo,
		Digest:    dig,
	}))

	return c.Next()

}

func (r Registry) validateManifestPath(c *fiber.Ctx) error {
	name := c.Params("+1")
	ref := c.Params("ref")
	if !r.nsRE.MatchString(name) {
		return fiber.NewError(fiber.StatusNotFound, "wrong name path")
	}

	var tag string
	var dig types.Digest
	if r.tagRE.MatchString(ref) {
		tag = ref
	} else if r.digRE.MatchString(ref) {
		sp := strings.Split(ref, ":")
		dig.Algo = sp[0]
		dig.Enc = sp[1]
	} else {
		return fiber.NewError(fiber.StatusNotFound, "wrong reference path")
	}

	ns, repo, err := parseName(name)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("failed parsing name: %s", err))
	}

	mid := types.ManifestID{
		Namespace: ns,
		Repo:      repo,
	}
	if tag != "" {
		mid.Tag = &tag
	} else {
		mid.Digest = &dig
	}

	c.SetUserContext(context.WithValue(c.UserContext(), midCtxKey{}, mid))

	return c.Next()
}

func parseName(name string) (string, string, error) {
	i := strings.LastIndex(name, "/")
	if i == -1 {
		return "", "", fmt.Errorf("name has no repo part")
	}
	ns := name[0:i]
	repo := name[i+1:]
	if ns == "" || repo == "" {
		return "", "", fmt.Errorf("namespace or repo empty")
	}

	return ns, repo, nil
}
