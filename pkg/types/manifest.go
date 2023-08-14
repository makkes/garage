// Copyright 2023 Max Jonas Werner
// SPDX-License-Identifier: GPL-3.0-or-later

package types

type ManifestID struct {
	Namespace, Repo string
	Tag             *string
	Digest          *Digest
}

func (m ManifestID) Ref() string {
	if m.Tag != nil {
		return *m.Tag
	}
	if m.Digest != nil {
		return m.Digest.String()
	}

	return ""
}
