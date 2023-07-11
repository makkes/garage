package types

import (
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"strings"
)

type SupportedAlgos string

const (
	AlgoSHA256 = SupportedAlgos("sha256")
	AlgoSHA512 = SupportedAlgos("sha512")
)

var digestCtors map[string]func() hash.Hash = map[string]func() hash.Hash{
	"sha256": sha256.New,
	"sha512": sha512.New,
}

type Digest struct {
	Algo, Enc string
}

func ParseDigest(s string) (Digest, error) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return Digest{}, fmt.Errorf("unexpected digest format %q", s)
	}

	if _, ok := digestCtors[parts[0]]; !ok {
		return Digest{}, fmt.Errorf("%s is an unsupported algorithm", parts[0])
	}

	return Digest{
		Algo: parts[0],
		Enc:  parts[1],
	}, nil
}

func (d Digest) String() string {
	return d.Algo + ":" + d.Enc
}

func NewDigest(alg SupportedAlgos, data io.Reader) (Digest, error) {
	var h hash.Hash

	ctor := digestCtors[string(alg)]
	if ctor == nil {
		return Digest{}, fmt.Errorf("unsupported algorithm %q requested", alg)
	}

	h = ctor()

	if _, err := io.Copy(h, data); err != nil {
		return Digest{}, fmt.Errorf("failed preparing hash: %w", err)
	}

	return Digest{
		Algo: string(alg),
		Enc:  fmt.Sprintf("%x", h.Sum(nil)),
	}, nil
}
