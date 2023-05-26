package http

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
)

const (
	InPrefix = iota
)

// type Request struct {
// 	Method   string
// 	Metadata []string

// 	RequestLine []byte
// 	Headers     []*HeaderKV
// 	Body        []byte

// 	isClear bool
// 	inBuf   bytes.Buffer // use for input request
// 	outBuf  bytes.Buffer // use for rewrite request
// }

/**
Request examples:

1. normal get request
```request
# tag: white
GET / HTTP/1.1
Host: waf-ce.chaitin.cn
```

2. normal post request
```request
# tag: black,sqli
POST /index.php HTTP/1.1
Host: waf-ce.chaitin.cn

id=1' or 1=1&submit=submit
```

3. abnormal post request
```request
# tag: black,abnormal
POST /index.php HTTP/1.1
Host: waf-ce.chaitin.cn
abnormal_headers

id=1&submit=submit
```

to be explore!!!
*/

type Request struct {
	Method       string           // http method
	CommentLines [][]byte         // all comment lines, for request's description or tag
	HeaderLines  [][]byte         // all request lines, include comment lines, header lines, body lines
	Headers      map[string][]int // record headers and it's line number
	Body         []byte           // all the request body, include smuggling things

	hasHost          bool // true if a request has `Host` header
	hasMultiHost     bool // true if a request has more than one `Host` header
	hasContentLength bool // true if a request has `Content-Length` header
	hasBody          bool // true if a request has request body

	inBuf  bytes.Buffer // use for input request
	outBuf bytes.Buffer // use for rewrite request
}

// SetHost set Host header to the target host, remove unnecessary hosts.
func (r *Request) SetHost(host string) {
	if hs, ok := r.Headers[HEADER_HOST]; ok {
		r.hasHost = true
		if len(hs) > 1 {
			r.hasMultiHost = true
		}

	} else {
		// not exist Host header
		r.AppendHeader(HEADER_HOST, host)
	}
}

// CalculateContentLength calculate body and set right Content-Length header, remove unnecessary content-length.
func (r *Request) CalculateContentLength() {
	foundContentLength := false
	bodyLength := len(r.Body)
	for k, v := range r.Headers {
		if bytes.EqualFold(v.Key, []byte("content-length")) {
			if !foundContentLength && bodyLength != 0 {
				foundContentLength = true
				declareLength, _ := strconv.Atoi(string(v.Value))
				if declareLength != bodyLength {
					r.Headers[k].Value = []byte(fmt.Sprintf("%d", bodyLength))
					r.isClear = false
				}
			} else {
				// remove other Content-Length header
				r.Headers = append(r.Headers[:k], r.Headers[k+1:]...)
				r.isClear = false
			}
		}
	}
	if !foundContentLength && bodyLength > 0 {
		r.AddHeader("Content-Length", fmt.Sprintf("%d", bodyLength))
	}
}

// GetHeaders get all key's value
func (r *Request) GetHeaders(key string) (values []string) {
	values = make([]string, 0)
	for _, v := range r.Headers {
		if bytes.EqualFold(v.Key, []byte(key)) {
			values = append(values, string(v.Value))
		}
	}
	return
}

// GetHeader get the first key value
func (r *Request) GetHeader(key string) (value string) {
	for _, v := range r.Headers {
		if bytes.EqualFold(v.Key, []byte(key)) {
			return string(v.Value)
		}
	}
	return ""
}

// SetHeader set the header key with value
func (r *Request) SetHeader(key string, value string) {
	foundKey := false
	for _, v := range r.Headers {
		if bytes.Equal(v.Key, []byte(key)) {
			foundKey = true
			if bytes.Equal(v.Value, []byte(value)) {
				continue
			}
			r.isClear = false
			kv := &HeaderKV{
				Key:   []byte(key),
				Value: []byte(value),
			}
			r.Headers = append(r.Headers, kv)
		}
	}
	if !foundKey {
		r.isClear = false
		r.AddHeader(key, value)
	}
}

// AppendHeader add a header key value
func (r *Request) AppendHeader(key string, value string) {
	i := len(r.HeaderLines)
	r.HeaderLines = append(r.HeaderLines, []byte(value))
	r.Headers[key] = []int{i}
}

// Len implements the Request buffer length method.
func (r *Request) Len() int {
	return r.outBuf.Len()
}

// Parse implements the Request Parse method.
func (r *Request) Parse(b []byte) (int, error) {
	n, err := r.inBuf.Read(b)
	if err != nil {
		return n, err
	}
	r.isClear = false
	return r.parse()
}

// Read implements the Request Read method.
func (r *Request) Write(b []byte) (int, error) {
	nwrite, err := r.inBuf.Write(b)
	if err != nil {
		return nwrite, err
	}
	r.isClear = false
	return r.parse()
}

// WriteTo implements the Request WriteTo method.
func (r *Request) WriteTo(w io.Writer) (int64, error) {
	r.reWrite()
	return r.outBuf.WriteTo(w)
}

// Read implements the Request Read method.
func (r *Request) Read(b []byte) (int, error) {
	r.reWrite()
	return r.outBuf.Read(b)
}

// ReadFrom implements the Request ReadFrom method.
func (r *Request) ReadFrom(in io.Reader) (int64, error) {
	nread, err := r.inBuf.ReadFrom(in)
	if err != nil {
		return nread, err
	}
	r.isClear = false
	nparse, err := r.parse()
	return int64(nparse), err
}

// String implements the Request String method.
func (r *Request) String() string {
	headerCnt := len(r.Headers)
	bodyLength := len(r.Body)

	return fmt.Sprintf("Request method: %s http headers number: %d body length: %d\nRequest Line: %s\n", r.Method, headerCnt, bodyLength, string(r.RequestLine))
}

func (r *Request) Dump() string {
	r.reWrite()
	return r.outBuf.String()
}

func (r *Request) reWrite() {
	if r.isClear {
		return
	}

	r.outBuf.Reset()

	r.outBuf.Write(r.RequestLine)
	for _, v := range r.Headers {
		r.outBuf.Write(v.Key)
		r.outBuf.WriteString(": ")
		r.outBuf.Write(v.Value)
		r.outBuf.WriteString("\r\n")
	}
	r.outBuf.WriteString("\r\n")
	if len(r.Body) > 0 {
		r.outBuf.Write(r.Body)
	}
	r.isClear = true
}

func (r *Request) ReadFile(file string) error {
	fp, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fp.Close()

	_, err = r.ReadFrom(fp)
	return err
}
