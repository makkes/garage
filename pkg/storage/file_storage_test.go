package storage_test

import (
	"errors"
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
	"github.com/makkes/garage/pkg/test/matchers"
	"github.com/makkes/garage/pkg/types"
)

func stringPtr(s string) *string {
	return &s
}

func TestError(t *testing.T) {
	g := NewWithT(t)

	err1 := storage.ErrNotFound{Err: fmt.Errorf("err 1")}
	err2 := storage.ErrNotFound{Err: fmt.Errorf("err 2")}

	is := errors.As(err1, &err2)
	g.Expect(is).To(BeTrue())
}

func TestTags(t *testing.T) {
	tests := []struct {
		name     string
		ns, repo string
		expErr   error
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
			expErr: storage.ErrNotFound{},
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

			if tt.expErr != nil {
				g.Expect(err).To(matchers.BeAssignableToError(tt.expErr))
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

func TestDeleteManifestWritesCorrectDataToDisk(t *testing.T) {
	g := NewWithT(t)

	manifest := `{"some":"manifest"}`
	dig, err := types.NewDigest(types.AlgoSHA256, strings.NewReader(manifest))
	g.Expect(err).NotTo(HaveOccurred(), "failed calculating digest")

	tests := []struct {
		name          string
		storeMid      types.ManifestID
		deleteMid     types.ManifestID
		expectedFiles []string
	}{
		{
			name: "delete by tag",
			storeMid: types.ManifestID{
				Namespace: "foo-ns",
				Repo:      "bar-repo",
				Tag:       stringPtr("baz-tag"),
				Digest:    &dig,
			},
			deleteMid: types.ManifestID{
				Namespace: "foo-ns",
				Repo:      "bar-repo",
				Tag:       stringPtr("baz-tag"),
				Digest:    &dig,
			},
			expectedFiles: []string{
				filepath.Join("_blobs", dig.String()),
				filepath.Join("foo-ns", "bar-repo", "_blobs", dig.String()),
				filepath.Join("foo-ns", "bar-repo", dig.String()),
			},
		},
		{
			name: "delete by digest",
			storeMid: types.ManifestID{
				Namespace: "foo-ns",
				Repo:      "bar-repo",
				Tag:       stringPtr("another-tag"),
				Digest:    &dig,
			},
			deleteMid: types.ManifestID{
				Namespace: "foo-ns",
				Repo:      "bar-repo",
				Tag:       nil,
				Digest:    &dig,
			},
			expectedFiles: []string{
				filepath.Join("_blobs", dig.String()),
				filepath.Join("foo-ns", "bar-repo", "_blobs", dig.String()),
				filepath.Join("foo-ns", "bar-repo", "_tags", "another-tag"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			// Given

			storeDir := t.TempDir()
			store, _ := storage.NewFileStorage(storeDir, logr.Discard())
			g.Expect(store.StoreManifest(tt.storeMid, strings.NewReader(manifest))).To(Succeed(), "storing manifest failed")

			// When

			g.Expect(store.DeleteManifest(tt.deleteMid)).To(Succeed(), "deleting manifest failed")

			// Then

			expectedFiles := make(map[string]bool, len(tt.expectedFiles))
			for _, ef := range tt.expectedFiles {
				expectedFiles[filepath.Join(storeDir, ef)] = false
			}
			g.Expect(filepath.WalkDir(storeDir, func(path string, d fs.DirEntry, err error) error {
				if _, has := expectedFiles[path]; !d.IsDir() && !has {
					return fmt.Errorf("unexpected non-dir encountered: %s. Expected: %v", path, tt.expectedFiles)
				}
				expectedFiles[path] = true
				return nil
			})).To(Succeed())

			for p, found := range expectedFiles {
				g.Expect(found).To(BeTrue(), "expected file %s to exist but it did not", p)
			}
		})
	}
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
