package storage

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/makkes/garage/pkg/types"
)

type MemStorage struct {
	manifests map[[sha256.Size]byte]types.Digest
	blobs     map[types.BlobID][]byte
}

var _ Storage = MemStorage{}

func NewMemStorage() MemStorage {
	return MemStorage{
		manifests: make(map[[sha256.Size]byte]types.Digest),
		blobs:     make(map[types.BlobID][]byte),
	}
}

func (m MemStorage) Tags(ns, repo string) ([]string, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m MemStorage) StartSession() (uuid.UUID, error) {
	return uuid.UUID{}, fmt.Errorf("not implemented")
}

func (m MemStorage) GetSessionInfo(_ uuid.UUID) (int64, error) {
	panic("not implemented")
}

func (m MemStorage) StoreSessionData(_ uuid.UUID, _ io.Reader, _ string) (int64, error) {
	panic("not implemented") // TODO: Implement
}

func (m MemStorage) CloseSession(_ uuid.UUID, bid types.BlobID) (types.Digest, error) {
	panic("not implemented") // TODO: Implement
}

func (m MemStorage) StoreBlob(bid types.BlobID, data io.Reader) (types.Digest, error) {
	b, err := io.ReadAll(data)
	if err != nil {
		return types.Digest{}, fmt.Errorf("failed reading data to store: %w", err)
	}
	m.blobs[bid] = b

	return types.NewDigest(types.AlgoSHA256, bytes.NewReader(b))
}

func (m MemStorage) FetchBlob(dig types.BlobID) (io.ReadCloser, BlobStat, error) {
	dat, ok := m.blobs[dig]
	if !ok {
		return nil, BlobStat{}, ErrNotFound{Err: fmt.Errorf("blob with digest %s not found", dig)}
	}

	return io.NopCloser(bytes.NewReader(dat)), BlobStat{Size: int64(len(dat))}, nil
}

func (m MemStorage) StoreManifest(id types.ManifestID, data io.Reader) error {
	if id.Digest == nil {
		return fmt.Errorf("can't store manifest without digest")
	}

	blob, err := io.ReadAll(data)
	if err != nil {
		return fmt.Errorf("failed reading input data: %w", err)
	}

	m.blobs[types.BlobID{
		Namespace: id.Namespace,
		Repo:      id.Repo,
		Digest:    *id.Digest,
	}] = blob

	dig := id.Digest

	m.manifests[hash(id)] = *dig

	id.Digest = nil
	m.manifests[hash(id)] = *dig

	id.Digest = dig
	id.Tag = nil
	m.manifests[hash(id)] = *dig

	return nil
}

func (m MemStorage) DeleteManifest(_ types.ManifestID) error {
	panic("not implemented") // TODO: Implement
}

func (m MemStorage) DeleteBlob(_ types.BlobID) error {
	panic("not implemented") // TODO: Implement
}

func (m MemStorage) FetchManifest(id types.ManifestID) (io.ReadCloser, error) {
	var dig types.Digest
	if id.Digest != nil {
		dig = *id.Digest
	} else {
		var ok bool
		dig, ok = m.manifests[hash(id)]
		if !ok {
			return nil, fmt.Errorf("manifest not found")
		}
	}

	rawMf, ok := m.blobs[types.BlobID{
		Namespace: id.Namespace,
		Repo:      id.Repo,
		Digest:    dig,
	}]
	if !ok {
		return nil, fmt.Errorf("manifest not found")
	}

	return io.NopCloser(bytes.NewReader(rawMf)), nil
}

func (m MemStorage) Has(id types.ManifestID) (bool, error) {
	_, ok := m.manifests[hash(id)]
	return ok, nil
}

func hash(m types.ManifestID) [sha256.Size]byte {
	var buf bytes.Buffer

	buf.WriteString(m.Namespace)
	buf.WriteString(m.Repo)

	if m.Tag != nil {
		buf.WriteString(*m.Tag)
	} else if m.Digest != nil {
		buf.WriteString(m.Digest.Algo)
		buf.WriteString(m.Digest.Enc)
	}

	return sha256.Sum256(buf.Bytes())
}
