package storage_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/makkes/garage/pkg/storage"
	"github.com/makkes/garage/pkg/types"
)

var impls = map[string]func(*testing.T) storage.Storage{
	"file storage": func(t *testing.T) storage.Storage {
		dir := t.TempDir()
		s, _ := storage.NewFileStorage(dir, logr.Discard())
		return s
	},
	"mem storage": func(_ *testing.T) storage.Storage {
		return storage.NewMemStorage()
	},
}

func TestFetchByTagAndDigestReturnsExpectedData(t *testing.T) {
	for impl, ctor := range impls {
		t.Run(impl, func(t *testing.T) {
			g := NewWithT(t)
			store := ctor(t)

			// Given

			dig := types.Digest{
				Algo: string(types.AlgoSHA256),
				Enc:  "7a38bf81f383f69433ad6e900d35b3e2385593f76a7b7ab5d4355b8ba41ee24b",
			}

			mdata := `{"foo":"bar"}`
			mid := types.ManifestID{
				Namespace: "foo-ns",
				Repo:      "bar-repo",
				Tag:       stringPtr("baz-tag"),
				Digest:    &dig,
			}
			g.Expect(store.StoreManifest(mid, strings.NewReader(mdata))).To(Succeed(), "storing manifest failed")

			// When (fetch by tag)

			mid.Digest = nil

			// Then

			g.Expect(store.Has(mid)).To(BeTrue(), "Has returned unexpected result for tag")

			rdr, err := store.FetchManifest(mid)
			g.Expect(err).NotTo(HaveOccurred(), "fetch failed for tag")
			g.Eventually(gbytes.BufferReader(rdr)).Should(gbytes.Say(mdata), "unexpected data in returned manifest from tag")

			// When (fetch by digest)

			mid.Digest = &dig
			mid.Tag = nil

			g.Expect(store.Has(mid)).To(BeTrue(), "Has returned unexpected result for digest")

			rdr, err = store.FetchManifest(mid)
			g.Expect(err).NotTo(HaveOccurred(), "fetch failed for digest")
			g.Eventually(gbytes.BufferReader(rdr)).Should(gbytes.Say(mdata), "unexpected data in returned manifest from")
		})
	}
}

func TestFetchBlobReturnsCorrectData(t *testing.T) {
	for impl, ctor := range impls {
		t.Run(impl, func(t *testing.T) {
			g := NewWithT(t)
			store := ctor(t)

			// Given

			bid := types.BlobID{
				Namespace: "foo",
				Repo:      "bar",
				Digest: types.Digest{
					Algo: string(types.AlgoSHA256),
					Enc:  "596f4162a52f315b2ad0fa53fd30a2769d02a41ed7439123790966eee4ceb5cd",
				},
			}
			blob := []byte{42, 42, 42}

			g.Expect(store.StoreBlob(bid, bytes.NewReader(blob))).To(Equal(bid.Digest), "storing blob failed")

			// When

			rdr, bs, err := store.FetchBlob(bid)

			// Then

			g.Expect(err).NotTo(HaveOccurred(), "FetchBlob returned unexpected error")
			g.Expect(bs.Size).To(Equal(int64(3)), "unexpected size in BlobStat")

			dat := make([]byte, 100)
			n, err := rdr.Read(dat)

			g.Expect(err).NotTo(HaveOccurred(), "unexpected error when reading blob data")
			g.Expect(n).To(Equal(3), "unexpected number of bytes read from blob reader")
			g.Expect(dat[:3]).To(Equal(blob))
		})
	}
}

func TestFetchBlobDoesntReturnBlobFromOtherRepo(t *testing.T) {
	for impl, ctor := range impls {
		t.Run(impl, func(t *testing.T) {
			g := NewWithT(t)
			store := ctor(t)

			// Given

			bid := types.BlobID{
				Namespace: "foo",
				Repo:      "bar",
				Digest: types.Digest{
					Algo: string(types.AlgoSHA256),
					Enc:  "596f4162a52f315b2ad0fa53fd30a2769d02a41ed7439123790966eee4ceb5cd",
				},
			}
			blob := []byte{42, 42, 42}

			g.Expect(store.StoreBlob(bid, bytes.NewReader(blob))).To(Equal(bid.Digest), "storing blob failed")

			// When

			bid.Repo = "another-one"
			blobRdr, _, err := store.FetchBlob(bid)

			// Then

			g.Expect(blobRdr).To(BeNil(), "blob reader should have been nil")
			g.Expect(errors.As(err, &storage.ErrNotFound{})).To(BeTrue(), "unexpected error returned")
		})
	}
}
