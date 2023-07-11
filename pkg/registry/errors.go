package registry

const (
	ERR_CODE_BLOB_UNKNOWN = "BLOB_UNKNOWN"
	ERR_MANIFEST_INVALID = "MANIFEST_INVALID"
)

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Errors []Error `json:"errors"`
}
