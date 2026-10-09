package ohmyhttp

const (
	jsonMediaType      = "application/json"
	formMediaType      = "application/x-www-form-urlencoded"
	multipartMediaType = "multipart/form-data"

	jsonContentType = jsonMediaType + "; charset=utf-8"
)

func isJSONMediaType(mediaType string) bool {
	return mediaType == jsonMediaType
}
