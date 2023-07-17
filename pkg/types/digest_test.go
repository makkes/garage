package types_test

import (
	"bytes"
	"testing"

	"github.com/opencontainers/go-digest"

	"github.com/makkes/garage/pkg/types"
)

func TestNewDigest(t *testing.T) {
	b := []byte("{\n\t\"author\": \"5rvjoBkDOo7323tG\",\n\t\"architecture\": \"amd64\",\n\t\"os\": \"linux\",\n\t\"rootfs\": {\n\t\t\"type\": \"layers\",\n\t\t\"diff_ids\": []\n\t}\n}")

	dig1 := digest.Canonical.FromBytes(b)
	dig2, err := types.NewDigest(types.AlgoSHA256, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("digesting failed: %s", err)
	}
	if dig1.String() != dig2.String() {
		t.Fatalf("%s != %s", dig1.String(), dig2.String())
	}

	t.Log(dig2.String())
}
