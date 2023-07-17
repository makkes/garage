package registry

const (
	ErrCodeBlobUnknown     = "BLOB_UNKNOWN"
	ErrCodeManifestInvalid = "MANIFEST_INVALID"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Errors []Error `json:"errors"`
}
