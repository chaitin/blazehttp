package http

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestRequest(t *testing.T) {
	s := `
GET /123 HTTP/1.1
Host: skynet.chaitin.net
Cookie: fake cookie

`
	buf := bytes.NewBufferString(s)
	req := new(Request)
	_, err := io.Copy(req, buf)
	if err != nil {
		t.Errorf("request read error: %s", err)
	}
	if !strings.EqualFold(req.GetHeader("Host"), "skynet.chaitin.net") {
		t.Errorf("want get Host: [%s], got: [%s]", "skynet.chaitin.net", req.GetHeader("Host"))
	}
	req.SetHost("1.2.3.4")
	if !strings.EqualFold(req.GetHeader("Host"), "1.2.3.4") {
		t.Errorf("want get Host: %s, got: %s", "1.2.3.4", req.GetHeader("Host"))
	}

	// for debug

	// fmt.Fprint(os.Stderr, req)
	// output:
	//
	// Request method: GET http version: HTTP/1.1 headers number: 2 body length: 0
	// Request Line: GET /123 HTTP/1.1

	// io.Copy(os.Stderr, req)
	// output:
	//
	// GET /123 HTTP/1.1
	// Host: 1.2.3.4
	// Cookie: fake cookie
}

func BenchmarkRequest(b *testing.B) {
	buf := bytes.NewBufferString("GET / HTTP/1.0\r\nHost: cookie.com\r\n\r\n")
	req := new(Request)

	for i := 0; i < b.N; i++ {
		_, _ = req.ReadFrom(buf)
	}
}
func TestRequestRawParse(t *testing.T) {
	buf := bytes.NewBufferString("GET / HTTP/1.0\r\nHost: cookie.com\r\n\r\n")
	req := new(Request)
	if _, err := req.ReadFrom(buf); err != nil {
		t.Error(err)
	}
}
