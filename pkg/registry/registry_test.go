package registry_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/makkes/garage/pkg/registry"
)

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

	req = httptest.NewRequest(http.MethodGet, expectedLoc.String(), nil)

	resp, err = r.App.Test(req)

	g.Expect(err).NotTo(HaveOccurred(), "test request failed unexpectedly")
	g.Expect(resp).To(HaveHTTPStatus(http.StatusOK), "manifest pull failed")
	g.Expect(resp).To(HaveHTTPBody(manifest), "manifest pull returned unexpected body")
}
