// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package registry_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/makkes/garage/pkg/features"
	"github.com/makkes/garage/pkg/registry"
)

func TestPushManifestWithLegacyHeader(t *testing.T) {
	g := NewWithT(t)

	r, _ := registry.New(
		registry.WithMemStorage(),
		registry.WithFeatures(features.Features{
			Flag: &[]string{
				features.SendLegacyDigestHeader,
			},
		}),
	)

	mt := "application/vnd.oci.image.manifest.v1+json"
	manifest := []byte(`{"mediaType":"` + mt + `"}`)

	req := httptest.NewRequest(http.MethodPut, "/v2/ns/repo/manifests/new-ref", bytes.NewReader(manifest))
	req.Header.Add("Content-Type", mt)

	resp, err := r.Test(req)

	g.Expect(err).NotTo(HaveOccurred(), "test request failed unexpectedly")
	g.Expect(resp.StatusCode).To(Equal(http.StatusCreated), "received unexpected status code")

	g.Expect(resp.Header["Docker-Content-Digest"]).To(Equal([]string{"sha256:0a1b17bf6d39f56897a7e8a056d930cf2bde38841a187aeb083d7487e2224573"}))
}

func TestPushAndPullManifest(t *testing.T) {
	g := NewWithT(t)

	r, _ := registry.New(
		registry.WithMemStorage(),
	)

	mt := "application/vnd.oci.image.manifest.v1+json"
	manifest := []byte(`{"mediaType":"` + mt + `"}`)

	req := httptest.NewRequest(http.MethodPut, "/v2/ns/repo/manifests/new-ref", bytes.NewReader(manifest))
	req.Header.Add("Content-Type", mt)

	resp, err := r.Test(req)

	g.Expect(err).NotTo(HaveOccurred(), "test request failed unexpectedly")
	g.Expect(resp.StatusCode).To(Equal(http.StatusCreated), "received unexpected status code")

	expectedLoc, err := url.Parse("/v2/ns/repo/manifests/new-ref")
	g.Expect(err).NotTo(HaveOccurred(), "failed parsing URL")
	g.Expect(resp.Location()).To(Equal(expectedLoc), "unexpected location header")
	g.Expect(resp.Header["Docker-Content-Digest"]).To(BeEmpty())

	req = httptest.NewRequest(http.MethodGet, expectedLoc.String(), nil)

	resp, err = r.App.Test(req)

	g.Expect(err).NotTo(HaveOccurred(), "test request failed unexpectedly")
	g.Expect(resp).To(HaveHTTPStatus(http.StatusOK), "manifest pull failed")
	g.Expect(resp).To(HaveHTTPBody(manifest), "manifest pull returned unexpected body")
}
