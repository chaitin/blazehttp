package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/chaitin/blazehttp/http"
	progressbar "github.com/schollz/progressbar/v3"
)

const (
	NoneTag = "none" // http file without tag
)

var (
	target           string // the target web site, example: http://192.168.0.1:8080
	glob             string // use glob expression to select multi files
	timeout          = 1000 // default 1000 ms
	statByCode       bool   // stat http response code
	statByCodeDetail bool   // show http response code detail
	statByTag        bool   // stat http request file tags
	statByTagDetail  bool   // stat http request file tags
	show_detail      bool   // show stat detail
)

func init() {
	flag.StringVar(&target, "t", "", "target website, example: http://192.168.0.1:8080")
	flag.StringVar(&glob, "g", "", "glob expression, example: *.http")
	flag.IntVar(&timeout, "timeout", 1000, "connection timeout, default 1000 ms")
	flag.BoolVar(&statByCode, "c", true, "stat http response code")
	flag.BoolVar(&statByCodeDetail, "C", false, "show http response code detail")
	flag.BoolVar(&statByTag, "s", true, "stat http request file tags")
	flag.BoolVar(&statByTagDetail, "S", false, "show http request file tags detail")
	flag.BoolVar(&show_detail, "d", false, "show stat detail")
	flag.Parse()
}

func connect(addr string, isHttps bool, timeout int) *net.Conn {
	var n net.Conn
	var err error

	if m, _ := regexp.MatchString(`.*(]:)|(:)[0-9]+$`, addr); !m {
		if isHttps {
			addr = fmt.Sprintf("%s:443", addr)
		} else {
			addr = fmt.Sprintf("%s:80", addr)
		}
	}

	retryCnt := 0
retry:
	if isHttps {
		n, err = tls.Dial("tcp", addr, nil)
	} else {
		n, err = net.Dial("tcp", addr)
	}
	if err != nil {
		retryCnt++
		if retryCnt < 4 {
			goto retry
		} else {
			return nil
		}
	}
	wDeadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
	rDeadline := time.Now().Add(time.Duration(timeout*2) * time.Millisecond)
	deadline := time.Now().Add(time.Duration(timeout*2) * time.Millisecond)
	_ = n.SetDeadline(deadline)
	_ = n.SetReadDeadline(rDeadline)
	_ = n.SetWriteDeadline(wDeadline)

	return &n
}

func statTags(tagStat map[string][]string, f string, metadatas []string) {
	if len(metadatas) == 0 {
		if _, ok := tagStat[NoneTag]; !ok {
			tagStat[NoneTag] = []string{f}
		} else {
			tagStat[NoneTag] = append(tagStat[NoneTag], f)
		}
		return
	}

	foundTag := false
	for _, line := range metadatas {
		l := strings.TrimSpace(line)
		if !strings.HasPrefix(l, "tag:") {
			continue
		}
		a := strings.SplitN(l, ":", 2)
		if len(a) < 2 {
			continue
		}
		t := strings.Split(a[1], ",")
		for _, v := range t {
			vv := strings.TrimSpace(v)
			foundTag = true
			if _, ok := tagStat[vv]; !ok {
				tagStat[vv] = []string{f}
			} else {
				tagStat[vv] = append(tagStat[vv], f)
			}
		}
	}
	if !foundTag {
		if _, ok := tagStat[NoneTag]; !ok {
			tagStat[NoneTag] = []string{f}
		} else {
			tagStat[NoneTag] = append(tagStat[NoneTag], f)
		}
	}
}

func main() {
	var codeStat map[int][]string
	var tagStat map[string][]string

	isHttps := false
	addr := target

	if strings.HasPrefix(target, "http") {
		u, _ := url.Parse(target)
		if u.Scheme == "https" {
			isHttps = true
		}
		addr = u.Host
	}

	fileList, err := filepath.Glob(glob)
	if err != nil || len(fileList) == 0 {
		fmt.Printf("cannot find http file")
		return
	}

	if statByCode {
		codeStat = make(map[int][]string)
	}
	if statByTag {
		tagStat = make(map[string][]string)
	}

	success := 0

	bar := progressbar.NewOptions64(
		int64(len(fileList)),
		progressbar.OptionSetDescription("sending"),
		progressbar.OptionSetWriter(os.Stderr),
		progressbar.OptionSetWidth(10),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionShowIts(),
		progressbar.OptionOnCompletion(func() {
			fmt.Fprint(os.Stderr, "\n")
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionUseANSICodes(true),
	)

	for _, f := range fileList {
		_ = bar.Add(1)
		req := new(http.Request)
		if err = req.ReadFile(f); err != nil {
			fmt.Printf("read request file: %s error: %s\n", f, err)
			continue
		}

		if statByTag {
			statTags(tagStat, f, req.Metadata)
		}

		req.SetHost(addr)
		// one http request one connection
		req.SetHeader("Connection", "close")
		// fix content length
		req.CalculateContentLength()

		conn := connect(addr, isHttps, timeout)
		if conn == nil {
			fmt.Printf("connect to %s failed!\n", addr)
			continue
		}
		nWrite, err := req.WriteTo(*conn)
		if err != nil {
			fmt.Printf("send request poc: %s length: %d error: %s", f, nWrite, err)
			continue
		}

		rsp := new(http.Response)
		if err = rsp.ReadConn(*conn); err != nil {
			fmt.Printf("read poc file: %s response, error: %s", f, err)
			continue
		}
		(*conn).Close()
		success++

		if statByCode {
			statusCode := rsp.GetStatusCode()
			if _, ok := codeStat[statusCode]; !ok {
				codeStat[statusCode] = []string{f}
			} else {
				codeStat[statusCode] = append(codeStat[statusCode], f)
			}
		}
	}

	fmt.Printf("Total http file: %d, success: %d failed: %d\n", len(fileList), success, (len(fileList) - success))

	if statByCode {
		fmt.Printf("Stat http response code\n\n")
		for k, v := range codeStat {
			fmt.Printf("Status code: %d hit: %d\n", k, len(v))
			if statByCodeDetail {
				for _, f := range v {
					fmt.Printf("- %s\n", f)
				}
			}
		}
		fmt.Println()
	}

	if statByTag {
		fmt.Printf("Stat http request tag\n\n")
		for k, v := range tagStat {
			fmt.Printf("tag: %s hit: %d\n", k, len(v))
			if statByTagDetail {
				for _, f := range v {
					fmt.Printf("- %s\n", f)
				}
			}
		}
		fmt.Println()
	}
}
