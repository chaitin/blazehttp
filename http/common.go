package http

type HeaderKV struct {
	Key   []byte
	Value []byte
}

const (
	HEADER_HOST           = "Host"
	HEADER_CONTENT_LENGTH = "Content-Length"
)
