package storage

import (
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/makkes/garage/pkg/types"
)

type ErrNotFound struct {
	Err error
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("blob or manifest not found: %s", e.Err)
}

type ErrSessionNotFound struct {
	Err error
}

func (e ErrSessionNotFound) Error() string {
	return fmt.Sprintf("session not found: %s", e.Err)
}

type ErrOutOfOrderChunk struct {
	expected, actual int64
}

func (e ErrOutOfOrderChunk) Error() string {
	return fmt.Sprintf("upload chunk out of order: expected %d but got %d", e.expected, e.actual)
}

type BlobStat struct {
	Size int64
}

type Storage interface {
	StoreBlob(types.BlobID, io.Reader) (types.Digest, error)
	FetchBlob(types.BlobID) (io.ReadCloser, BlobStat, error)
	DeleteBlob(types.BlobID) error

	StartSession() (uuid.UUID, error)
	GetSessionInfo(uuid.UUID) (int64, error)
	StoreSessionData(uuid.UUID, io.Reader, string) (int64, error)
	CloseSession(uuid.UUID, types.BlobID) (types.Digest, error)

	StoreManifest(types.ManifestID, io.Reader) error
	FetchManifest(types.ManifestID) (io.ReadCloser, error)
	Has(types.ManifestID) (bool, error)
	DeleteManifest(types.ManifestID) error

	Tags(ns, repo string) ([]string, error)
}
