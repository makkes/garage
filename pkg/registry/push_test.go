package registry_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/makkes/garage/pkg/registry"
)

func TestPushManifests(t *testing.T) {
	tests := []struct {
		name            string
		ref             string
		body            []byte
		maxManifestSize int64
		contentType     string
		expStatusCode   int
		expLocation     string
	}{
		{
			name:            "missing body",
			body:            nil,
			maxManifestSize: 10,
			expStatusCode:   http.StatusBadRequest,
		},
		{
			name:            "body too large",
			maxManifestSize: 10,
			body:            []byte(`{"a":"fffffffffffffffffff"}`),
			expStatusCode:   http.StatusRequestEntityTooLarge,
		},
		{
			name:            "invalid manifest format",
			maxManifestSize: 20,
			body:            []byte(`this is not JSON`),
			expStatusCode:   http.StatusBadRequest,
		},
		{
			name:            "missing Content-Type header",
			maxManifestSize: 20,
			body:            []byte(`{"mediaType":"a/b"}`),
			expStatusCode:   http.StatusBadRequest,
		},
		{
			name:            "Content-Type and mediaType mismatch",
			maxManifestSize: 20,
			body:            []byte(`{"mediaType":"a/b"}`),
			contentType:     "c/d",
			expStatusCode:   http.StatusBadRequest,
		},
		{
			name:            "wrong ref format",
			ref:             "-wrong:digest",
			body:            []byte(`{"mediaType":"foobar"}`),
			maxManifestSize: 40,
			contentType:     "foobar",
			expStatusCode:   http.StatusNotFound,
		},
		{
			name:            "push by tag",
			ref:             "v1.0.0",
			body:            []byte(`{"mediaType":"foo/bar"}`),
			maxManifestSize: 99,
			contentType:     "foo/bar",
			expStatusCode:   http.StatusCreated,
			expLocation:     "/v2/ns/repo/manifests/v1.0.0",
		},
		{
			name:            "push by digest",
			ref:             "sha256:foobar",
			body:            []byte(`{"mediaType":"foo/bar"}`),
			maxManifestSize: 99,
			contentType:     "foo/bar",
			expStatusCode:   http.StatusCreated,
			expLocation:     "/v2/ns/repo/manifests/sha256:foobar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			ref := "new-ref"
			if tt.ref != "" {
				ref = tt.ref
			}
			r, _ := registry.New(
				registry.WithMaxManifestBytes(tt.maxManifestSize),
				registry.WithMemStorage(),
			)
			req := httptest.NewRequest(http.MethodPut, "/v2/ns/repo/manifests/"+ref, bytes.NewReader(tt.body))
			if tt.contentType != "" {
				req.Header.Add("content-type", tt.contentType)
			}

			resp, err := r.App.Test(req)

			g.Expect(err).NotTo(HaveOccurred(), "test request failed unexpectedly")
			g.Expect(resp).To(HaveHTTPStatus(tt.expStatusCode))
			g.Expect(resp).To(HaveHTTPHeaderWithValue("location", tt.expLocation))
		})
	}
}
