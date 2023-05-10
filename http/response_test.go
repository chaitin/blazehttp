package http

import (
	"bytes"
	"io"
	"testing"
)

func TestResponse(t *testing.T) {
	s := `HTTP/1.1 200 OK
Server: nginx/1.18.0 (Ubuntu)
Date: Tue, 11 Apr 2023 10:03:46 GMT
Content-Type: text/html; charset=utf-8
Content-Length: 108
Connection: keep-alive

[竞品测试平台测试响应](sha256): 16d2097fa99913ffd2f9c300f532a91b6a1275c11b636e325863a681acb8098e
`
	buf := bytes.NewBufferString(s)
	rsp := new(Response)
	parseLength, err := io.Copy(rsp, buf)
	if err != nil {
		t.Errorf("response read error: %s", err)
	}
	if int(parseLength) != rsp.Len() {
		t.Errorf("parse response error: parse length: %d, response length: %d", parseLength, rsp.Len())
	}
	if !bytes.Equal(rsp.StatusCode, []byte("200")) {
		t.Errorf("want get Host: %s, got: %s", "200", rsp.StatusCode)
	}

	// for debug

	// for k, v := range rsp.Headers {
	// 	fmt.Fprintf(os.Stderr, "%d. [%s]:[%s]\n", k, v.Key, v.Value)
	// }

	// fmt.Fprint(os.Stderr, rsp)
	// output:
	//
	// Response status code: 200 http version: HTTP/1.1 headers number: 5 body length: 107
	// Response Line: HTTP/1.1 200 OK

	// io.Copy(os.Stderr, rsp)
	// output:
	//
	// HTTP/1.1 200 OK
	// Server: nginx/1.18.0 (Ubuntu)
	// Date: Tue, 11 Apr 2023 10:03:46 GMT
	// Content-Type: text/html; charset=utf-8
	// Content-Length: 108
	// Connection: keep-alive

	// [竞品测试平台测试响应](sha256): 16d2097fa99913ffd2f9c300f532a91b6a1275c11b636e325863a681acb8098e
}
