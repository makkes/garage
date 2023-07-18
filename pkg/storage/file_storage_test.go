package storage_test

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"

	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

func stringPtr(s string) *string {
	return &s
}

func TestTags(t *testing.T) {
	tests := []struct {
		name     string
		ns, repo string
		expErr   string
		expTags  []string
	}{
		{
			name:    "listing all tags succeeds",
			ns:      "namespace",
			repo:    "repo",
			expTags: []string{"test1"},
		},
		{
			name:   "listing tags in unknown repo fails",
			ns:     "does-not-exist",
			repo:   "repo",
			expErr: "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Given

			storeDir := "testdata/tags"
			store, _ := storage.NewFileStorage(storeDir, logr.Discard())

			// When

			tags, err := store.Tags(tt.ns, tt.repo)

			// Then

			if tt.expErr != "" {
				g.Expect(err).To(MatchError(ContainSubstring(tt.expErr)))
			}
			g.Expect(tags).To(Equal(tt.expTags), "unexpected tag list received")
		})
	}
}

func TestStoreManifestFailsWithWrongDigest(t *testing.T) {
	g := NewWithT(t)

	// Given

	storeDir := t.TempDir()
	store, _ := storage.NewFileStorage(storeDir, logr.Discard())
	dig, err := types.NewDigest("sha256", strings.NewReader("wrong-digest"))
	g.Expect(err).NotTo(HaveOccurred(), "digest creation failed")

	mid := types.ManifestID{
		Namespace: "foo-ns",
		Repo:      "bar-repo",
		Tag:       stringPtr("baz-tag"),
		Digest:    &dig,
	}

	manifest := `{"some":"manifest"}`

	// When

	g.Expect(store.StoreManifest(mid, strings.NewReader(manifest))).
		To(
			MatchError(ContainSubstring("digests don't match")),
			"storing manifest should have failed",
		)

	expectedFiles := []string{
		filepath.Join(storeDir, "_blobs", "sha256:12fa8bca3fe74bc436b1068c5bda9c82f3f4d3583a3ca1ded704cb43968ed9d4"),
	}
	g.Expect(filepath.WalkDir(storeDir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && !contains(expectedFiles, path) {
			return fmt.Errorf("unexpected non-dir encountered: %s. Expected: %s", path, expectedFiles)
		}
		return nil
	})).To(Succeed())
}

func TestStoreManifestWritesCorrectDataToDisk(t *testing.T) {
	g := NewWithT(t)

	// Given

	storeDir := t.TempDir()
	store, _ := storage.NewFileStorage(storeDir, logr.Discard())
	dig := types.Digest{
		Algo: string(types.AlgoSHA256),
		Enc:  "7a38bf81f383f69433ad6e900d35b3e2385593f76a7b7ab5d4355b8ba41ee24b",
	}

	mid := types.ManifestID{
		Namespace: "foo-ns",
		Repo:      "bar-repo",
		Tag:       stringPtr("baz-tag"),
		Digest:    &dig,
	}

	manifest := `{"foo":"bar"}`

	// When

	g.Expect(store.StoreManifest(mid, strings.NewReader(manifest))).To(Succeed(), "storing manifest failed")

	// Then

	manifestPath := filepath.Join(storeDir, mid.Namespace, mid.Repo, "_tags", *mid.Tag)
	f, err := os.Open(manifestPath)
	g.Expect(err).NotTo(HaveOccurred(), "failed opening stored manifest file")
	defer f.Close()

	dataFromDisk, err := io.ReadAll(f)
	g.Expect(err).NotTo(HaveOccurred(), "failed reading stored manifest file")

	g.Expect(dataFromDisk).To(Equal([]byte(dig.String())), "unexpected file content")

	// Make sure no other files were written

	expectedFiles := []string{
		filepath.Join(storeDir, "_blobs", dig.String()),
		filepath.Join(storeDir, mid.Namespace, mid.Repo, "_blobs", dig.String()),
		filepath.Join(storeDir, mid.Namespace, mid.Repo, mid.Digest.String()),
		manifestPath,
	}
	g.Expect(filepath.WalkDir(storeDir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && !contains(expectedFiles, path) {
			return fmt.Errorf("unexpected non-dir encountered: %s. Expected: %s", path, expectedFiles)
		}
		return nil
	})).To(Succeed())
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func TestWriteManifestDoesntWriteAnythingIfDigestMissing(t *testing.T) {
	g := NewWithT(t)

	dir := t.TempDir()
	store, _ := storage.NewFileStorage(dir, logr.Discard())

	mid := types.ManifestID{
		Namespace: "foo-ns",
		Repo:      "bar-repo",
		Tag:       stringPtr("baz-tag"),
	}

	g.Expect(store.StoreManifest(mid, nil)).NotTo(Succeed(), "storing manifest should have failed")

	g.Expect(filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return fmt.Errorf("unexpected non-dir encountered: %s", path)
		}
		return nil
	})).To(Succeed())
}
