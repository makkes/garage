// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package types

type BlobID struct {
	Namespace, Repo string
	Digest          Digest
}
