package http

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strconv"
)

type Response struct {
	Version string

	StatusCode []byte
	Reason     []byte
	StatusLine []byte
	Headers    []*HeaderKV
	Body       []byte

	buf bytes.Buffer
}

func (r *Response) GetStatusCode() int {
	code, err := strconv.Atoi(string(r.StatusCode))
	if err != nil {
		return -1
	}
	return code
}

// Len implements the Response buffer length method.
func (r *Response) Len() int {
	return r.buf.Len()
}

// Parse implements the Response Parse method.
func (r *Response) Parse(b []byte) (int, error) {
	n, err := r.buf.Read(b)
	if err != nil {
		return n, err
	}
	return r.parse()
}

// Read implements the Response Read method.
func (r *Response) Write(b []byte) (int, error) {
	nwrite, err := r.buf.Write(b)
	if err != nil {
		return nwrite, err
	}

	return r.parse()
}

// WriteTo implements the Response WriteTo method.
func (r *Response) WriteTo(w io.Writer) (n int64, err error) {
	return r.buf.WriteTo(w)
}

// Read implements the Response Read method.
func (r *Response) Read(b []byte) (int, error) {
	return r.buf.Read(b)
}

// ReadFrom implements the Response ReadFrom method.
func (r *Response) ReadFrom(in io.Reader) (int64, error) {
	nread, err := r.buf.ReadFrom(in)
	if err != nil {
		return nread, err
	}

	n, err := r.parse()

	return int64(n), err
}

// Close implements the Response Read method.
func (r *Response) Close() error {
	return nil
}

// String implements the Response String method.
func (r *Response) String() string {
	headerCnt := len(r.Headers)
	bodyLength := len(r.Body)
	return fmt.Sprintf("Response status code: %s http version: %s headers number: %d body length: %d\nResponse Line: %s\n", string(r.StatusCode), r.Version, headerCnt, bodyLength, string(r.StatusLine))
}

func (r *Response) ReadConn(conn net.Conn) (err error) {
	buf := new(bytes.Buffer)
	b := make([]byte, 1024)

	for {
		nread, err := conn.Read(b)
		if err != nil {
			break
		}
		if nread == 0 {
			break
		}
		buf.Write(b)
		if nread < cap(b) {
			break
		}
	}

	_, err = r.ReadFrom(buf)
	if err != nil {
		return err
	}

	return nil
}
