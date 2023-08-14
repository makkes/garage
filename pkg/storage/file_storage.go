// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/go-logr/logr"
	"github.com/google/uuid"

	"github.com/makkes/garage/pkg/types"
)

const (
	blobDirName       = "_blobs"
	tagDirName        = "_tags"
	contentRangeRegex = `^([0-9]+)-([0-9]+)$`
)

type FileStorage struct {
	baseDir string
	log     logr.Logger
	crRE    *regexp.Regexp
}

var _ Storage = FileStorage{}

func NewFileStorage(baseDir string, log logr.Logger) (FileStorage, error) {
	if err := ensureDir(filepath.Join(baseDir, blobDirName)); err != nil {
		return FileStorage{}, fmt.Errorf("failed ensuring base dir: %w", err)
	}

	return FileStorage{
		baseDir: baseDir,
		log:     log,
		crRE:    regexp.MustCompile(contentRangeRegex),
	}, nil
}

func (fs FileStorage) Tags(ns, repo string) ([]string, error) {
	p := filepath.Join(fs.baseDir, ns, repo, tagDirName)
	_, err := os.Stat(p)
	if err != nil {
		retErr := fmt.Errorf("failed checking tag dir: %w", err)
		if os.IsNotExist(err) {
			return nil, ErrNotFound{Err: retErr}
		}
		return nil, retErr
	}

	tags, err := os.ReadDir(p)
	if err != nil {
		return nil, fmt.Errorf("failed listing tag dir: %w", err)
	}

	res := make([]string, len(tags))
	for idx, f := range tags {
		res[idx] = f.Name()
	}

	return res, nil
}

func (fs FileStorage) StoreManifest(mid types.ManifestID, data io.Reader) (retErr error) {
	var rollbacks []func() error
	defer func() {
		if retErr == nil {
			return // no error => no rollbacks
		}
		for _, rb := range rollbacks {
			if err := rb(); err != nil {
				fs.log.Error(err, "failed performing rollback")
			}
		}
	}()

	if mid.Digest == nil {
		retErr = fmt.Errorf("digest cannot be nil when storing manifest")
		return
	}

	bid := types.BlobID{
		Namespace: mid.Namespace,
		Repo:      mid.Repo,
		Digest:    *mid.Digest,
	}
	dig, err := fs.StoreBlob(bid, data)
	if err != nil {
		retErr = fmt.Errorf("failed storing manifest file: %w", err)
		return
	}

	rollbacks = append(rollbacks, func() error {
		bid.Digest = dig
		return fs.DeleteBlob(bid)
	})

	if dig != *mid.Digest {
		retErr = fmt.Errorf("digests don't match: provided: %s, expected: %s", mid.Digest, dig)
		return
	}

	p := filepath.Join(fs.baseDir, mid.Namespace, mid.Repo)
	if err := ensureDir(p); err != nil {
		retErr = fmt.Errorf("failed ensuring repository directory: %w", err)
		return
	}

	tmpF, err := os.CreateTemp(p, ".")
	if err != nil {
		retErr = fmt.Errorf("failed creating temp file: %w", err)
		return
	}
	defer tmpF.Close()
	defer os.Remove(tmpF.Name())

	if _, err := tmpF.WriteString(mid.Digest.String()); err != nil {
		retErr = fmt.Errorf("failed writing manifest link file: %w", err)
		return
	}

	tmpF.Close()

	fn, err := fs.getFilename(mid)
	if err != nil {
		retErr = fmt.Errorf("failed deriving manifest file name: %w", err)
		return
	}

	if err := ensureDir(filepath.Dir(fn)); err != nil {
		retErr = fmt.Errorf("failed ensuring tag directory: %w", err)
		return
	}

	if err := os.Rename(tmpF.Name(), fn); err != nil {
		retErr = fmt.Errorf("failed creating tag manifest file: %w", err)
		return
	}

	rollbacks = append(rollbacks, func() error {
		return os.Remove(fn)
	})

	if err := os.WriteFile(filepath.Join(p, mid.Digest.String()), []byte(mid.Digest.String()), 0600); err != nil {
		retErr = fmt.Errorf("failed creating digest manifest file: %w", err)
		return
	}

	return nil
}

func (fs FileStorage) DeleteManifest(mid types.ManifestID) error {
	fn, err := fs.getFilename(mid)
	if err != nil {
		return fmt.Errorf("failed deriving manifest file name: %w", err)
	}

	return os.Remove(fn)
}

func (fs FileStorage) DeleteBlob(bid types.BlobID) error {
	return os.Remove(filepath.Join(fs.baseDir, bid.Namespace, bid.Repo, blobDirName, bid.Digest.String()))
}

func ensureDir(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if mkdErr := os.MkdirAll(path, 0750); mkdErr != nil {
				return fmt.Errorf("failed creating directory %q: %w", path, mkdErr)
			}
		} else {
			return fmt.Errorf("failed checking directory: %w", err)
		}
	}

	return nil
}

func (fs FileStorage) getFilename(mid types.ManifestID) (string, error) {
	switch {
	case mid.Tag != nil:
		return filepath.Join(fs.baseDir, mid.Namespace, mid.Repo, tagDirName, *mid.Tag), nil
	case mid.Digest != nil:
		return filepath.Join(fs.baseDir, mid.Namespace, mid.Repo, mid.Digest.String()), nil
	default:
		return "", fmt.Errorf("neither tag nor digest set for manifest")
	}
}

func (fs FileStorage) Has(mid types.ManifestID) (bool, error) {
	fname, err := fs.getFilename(mid)
	if err != nil {
		return false, fmt.Errorf("failed deriving manifest file name: %w", err)
	}

	fi, err := os.Stat(fname)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed verifying manifest file: %w", err)
	}

	if !fi.Mode().IsRegular() {
		return false, fmt.Errorf("manifest file is not a regular file")
	}

	return true, nil
}

func (fs FileStorage) FetchManifest(mid types.ManifestID) (io.ReadCloser, error) {
	fname, err := fs.getFilename(mid)
	if err != nil {
		return nil, fmt.Errorf("failed deriving manifest file name: %w", err)
	}
	linkBytes, err := os.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("failed reading manifest link: %w", err)
	}

	dig, err := types.ParseDigest(string(linkBytes))
	if err != nil {
		return nil, fmt.Errorf("failed parsing digest: %w", err)
	}

	b, _, err := fs.FetchBlob(types.BlobID{
		Namespace: mid.Namespace,
		Repo:      mid.Repo,
		Digest:    dig,
	})
	return b, err
}

func (fs FileStorage) StartSession() (uuid.UUID, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed generating session ID: %w", err)
	}

	tmpF, err := os.Create(filepath.Join(fs.baseDir, blobDirName, "_"+id.String()))
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("failed creating temp file: %w", err)
	}
	tmpF.Close()

	return id, nil
}

func (fs FileStorage) GetSessionInfo(id uuid.UUID) (int64, error) {
	fi, err := os.Stat(filepath.Join(fs.baseDir, blobDirName, "_"+id.String()))
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrSessionNotFound{Err: err}
		}
		return 0, fmt.Errorf("failed checking session file: %w", err)
	}

	return fi.Size(), nil
}

func (fs FileStorage) parseRange(s string) (int64, int64, error) {
	if s == "" {
		return 0, 0, nil
	}

	matches := fs.crRE.FindStringSubmatch(s)
	if len(matches) != 3 {
		return 0, 0, fmt.Errorf("range string %q doesn't match expected format %q", s, contentRangeRegex)
	}

	var start, end int64
	var err error
	start, err = strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed parsing start of range: %w", err)
	}
	end, err = strconv.ParseInt(matches[2], 10, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed parsing end of range: %w", err)
	}

	return start, end, nil
}

func (fs FileStorage) StoreSessionData(id uuid.UUID, in io.Reader, cr string) (int64, error) {
	p := filepath.Join(fs.baseDir, blobDirName, "_"+id.String())

	crs, cre, err := fs.parseRange(cr)
	if err != nil {
		return 0, fmt.Errorf("failed parsing content-range: %w", err)
	}

	fi, err := os.Stat(p)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrSessionNotFound{Err: err}
		}
		return 0, fmt.Errorf("failed checking session file: %w", err)
	}

	if crs >= 0 && cre > 0 && crs != fi.Size() {
		return 0, ErrOutOfOrderChunk{
			expected: fi.Size() + 1,
			actual:   crs,
		}
	}

	tmpF, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, ErrSessionNotFound{Err: err}
		}
		return 0, fmt.Errorf("failed opening temp file: %w", err)
	}
	defer tmpF.Close()

	n, err := io.Copy(tmpF, in)
	if err != nil {
		return 0, fmt.Errorf("failed writing session data: %w", err)
	}

	fs.log.V(7).Info("wrote data to session", "session", id, "bytes", n)

	return n - 1, nil
}

func (fs FileStorage) CloseSession(id uuid.UUID, bid types.BlobID) (types.Digest, error) {
	p := filepath.Join(fs.baseDir, blobDirName, "_"+id.String())
	var res types.Digest

	tmpF, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		if os.IsNotExist(err) {
			return res, ErrSessionNotFound{Err: err}
		}
		return res, fmt.Errorf("failed opening temp file: %w", err)
	}
	defer tmpF.Close()

	return fs.finalizeBlob(tmpF.Name(), bid)
}

func (fs FileStorage) FetchBlob(bid types.BlobID) (io.ReadCloser, BlobStat, error) {
	_, err := os.Stat(filepath.Join(fs.baseDir, bid.Namespace, bid.Repo, blobDirName, bid.Digest.String()))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, BlobStat{}, ErrNotFound{Err: err}
		}
		return nil, BlobStat{}, fmt.Errorf("failed finding blob link: %w", err)
	}

	fi, err := os.Stat(filepath.Join(fs.baseDir, blobDirName, bid.Digest.String()))
	if err != nil {
		return nil, BlobStat{}, fmt.Errorf("failing gathering blob info from filesystem: %w", err)
	}

	bs := BlobStat{
		Size: fi.Size(),
	}
	rdr, err := os.OpenFile(filepath.Join(fs.baseDir, blobDirName, bid.Digest.String()), os.O_RDONLY, 0)
	return rdr, bs, err
}

func (fs FileStorage) finalizeBlob(tmpF string, bid types.BlobID) (types.Digest, error) {
	f, err := os.Open(tmpF)
	if err != nil {
		return types.Digest{}, fmt.Errorf("failed opening blob file for digesting: %w", err)
	}
	defer f.Close()

	dig, err := types.NewDigest(types.AlgoSHA256, f)
	if err != nil {
		return types.Digest{}, fmt.Errorf("failed calculating content digest: %w", err)
	}

	blobFileName := filepath.Join(fs.baseDir, blobDirName, dig.String())

	if err := os.Rename(tmpF, blobFileName); err != nil {
		return types.Digest{}, fmt.Errorf("failed creating final blob file: %w", err)
	}

	blobDir := filepath.Join(fs.baseDir, bid.Namespace, bid.Repo, blobDirName)
	if err := ensureDir(blobDir); err != nil {
		return types.Digest{}, fmt.Errorf("failed ensuring repo blob directory: %w", err)
	}

	f, err = os.OpenFile(filepath.Join(blobDir, dig.String()), os.O_CREATE, 0640)
	if err != nil {
		return types.Digest{}, fmt.Errorf("failed creating blob link: %w", err)
	}
	f.Close()

	return dig, nil
}

func (fs FileStorage) StoreBlob(bid types.BlobID, data io.Reader) (types.Digest, error) {
	p := filepath.Join(fs.baseDir, blobDirName)
	if err := ensureDir(p); err != nil {
		return types.Digest{}, fmt.Errorf("failed ensuring blob directory: %w", err)
	}

	tmpF, err := os.CreateTemp(p, ".")
	if err != nil {
		return types.Digest{}, fmt.Errorf("failed creating temp file: %w", err)
	}
	defer os.Remove(tmpF.Name())

	if _, err := io.Copy(tmpF, data); err != nil {
		return types.Digest{}, fmt.Errorf("failed writing blob data to file: %w", err)
	}

	return fs.finalizeBlob(tmpF.Name(), bid)
}
