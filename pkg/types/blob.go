package types

type BlobID struct {
	Namespace, Repo string
	Digest          Digest
}
