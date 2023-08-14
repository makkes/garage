// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	. "github.com/onsi/gomega"

	"github.com/makkes/garage/pkg/registry"
	"github.com/makkes/garage/pkg/storage"
)

func TestPullManifest(t *testing.T) {
	tests := []struct {
		name                string
		path                string
		methods             []string
		reqHdrs             map[string]string
		expectedStatusCode  int
		expectedContentType string
	}{
		{
			name:                "happy path",
			methods:             []string{http.MethodGet, http.MethodHead},
			path:                "/v2/ns/repo/manifests/ref",
			reqHdrs:             map[string]string{"Accept": "text/plain"},
			expectedStatusCode:  http.StatusOK,
			expectedContentType: "text/plain",
		},
		{
			name:               "unknown namespace",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/does-not-exist/repo/manifests/ref",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "unknown repo",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/ns/does-not-exist/manifests/ref",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "unknown manifest ref",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/ns/repo/manifests/does-not-exist",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid namespace prefix",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/_ns/repo/manifests/ref",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid namespace suffix",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/ns-/repo/manifests/ref",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "invalid ref prefix",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/ns/repo/manifests/-ref",
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:               "non-matching content-type",
			methods:            []string{http.MethodGet, http.MethodHead},
			path:               "/v2/ns/repo/manifests/ref",
			reqHdrs:            map[string]string{"Accept": "foo/bar"},
			expectedStatusCode: http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			s, err := storage.NewFileStorage("./testdata/registry", logr.Discard())
			g.Expect(err).NotTo(HaveOccurred(), "failed initializing file storage backend")

			stdr.SetVerbosity(10)

			r, _ := registry.New(
				registry.WithFileStorage(s),
				registry.WithLogger(stdr.New(log.New(os.Stdout, "", log.LstdFlags))),
			)

			for _, m := range tt.methods {
				req := httptest.NewRequest(m, tt.path, nil)
				for k, v := range tt.reqHdrs {
					req.Header.Add(k, v)
				}

				resp, err := r.App.Test(req)

				g.Expect(err).NotTo(HaveOccurred(), "test request failed unexpectedly")
				g.Expect(resp).To(HaveHTTPStatus(tt.expectedStatusCode))
				if tt.expectedContentType != "" {
					g.Expect(resp).
						To(HaveHTTPHeaderWithValue("content-type", tt.expectedContentType), "unexpected content-type")
				}
			}
		})
	}
}
