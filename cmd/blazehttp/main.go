package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	blazehttp "github.com/chaitin/blazehttp/http"
	progressbar "github.com/schollz/progressbar/v3"
)

const (
	NoneTag = "none" // http file without tag
)

var (
	target            string // the target web site, example: http://192.168.0.1:8080
	glob              string // use glob expression to select multi files
	timeout           = 1000 // default 1000 ms
	mHost             string // modify host header
	thread            = 2    // send request thread
	requestPerSession bool   // send request per session
)

func init() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./blazehttp -t <url>")
		os.Exit(1)
	}
	flag.StringVar(&target, "t", "", "target website, example: http://192.168.0.1:8080")
	flag.StringVar(&glob, "g", "./testcases/", "glob expression, example: *.http")
	flag.IntVar(&timeout, "timeout", 1000, "connection timeout, default 1000 ms")
	flag.IntVar(&thread, "thread", 2, "request thread, default 2")
	flag.StringVar(&mHost, "H", "", "modify host header")
	flag.BoolVar(&requestPerSession, "rps", true, "send request per session")
	flag.Parse()
	if url, err := url.Parse(target); err != nil || url.Scheme == "" || url.Host == "" {
		fmt.Println("invalid target url, example: http://chaitin.com:9443")
		os.Exit(1)
	}
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
		n, err = tls.Dial("tcp", addr, &tls.Config{InsecureSkipVerify: true})
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

func getNomalStatusCode(url string, mHost string) (statusCode int, conErr error) {
	isHttps := strings.HasPrefix(url, "https")
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	req, err := http.NewRequest("GET", url, nil)

	if isHttps {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client = &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
			Transport: tr,
		}
	}
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %s", err)
	}
	if mHost != "" {
		req.Host = mHost
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.9999.999 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("HTTP 请求发生错误:", err)
		return 0, err
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode
	return
}

func getAllFiles(path string) ([]string, error) {
	var files []string

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, filePath)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func work(addr string, isHttps bool, blockStatusCode int, f string) (bool, bool, int64, error) {
	req := new(blazehttp.Request)

	if err := req.ReadFile(f); err != nil {
		return false, false, 0, fmt.Errorf("read request file: %s error: %s", f, err)
	}
	if mHost != "" {
		// 修改host header
		req.SetHost(mHost)
	} else {
		// 不修改会导致域名备案拦截
		req.SetHost(addr)
	}

	if requestPerSession {
		// one http request one connection
		req.SetHeader("Connection", "close")
	}

	req.CalculateContentLength()

	start := time.Now()
	conn := connect(addr, isHttps, timeout)
	if conn == nil {
		return false, false, 0, fmt.Errorf("connect to %s failed", addr)
	}
	nWrite, err := req.WriteTo(*conn)
	if err != nil {
		return false, false, 0, fmt.Errorf("send request poc: %s length: %d error: %s", f, nWrite, err)
	}

	rsp := new(blazehttp.Response)
	if err = rsp.ReadConn(*conn); err != nil {
		return false, false, 0, fmt.Errorf("read poc file: %s response, error: %s", f, err)
	}
	(*conn).Close()
	isWhite := false // black case
	if strings.HasSuffix(f, "white") {
		isWhite = true // white case
	}
	isPass := true
	code := rsp.GetStatusCode()
	if code == blockStatusCode {
		isPass = false
	}
	return isWhite, isPass, time.Since(start).Nanoseconds(), nil
}

func main() {

	isHttps := false
	addr := target

	if strings.HasPrefix(target, "http") {
		u, _ := url.Parse(target)
		if u.Scheme == "https" {
			isHttps = true
		}
		addr = u.Host
	}

	fileList, err := getAllFiles(glob)
	if err != nil || len(fileList) == 0 {
		fmt.Printf("cannot find http file")
		return
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

	TP := 0
	TN := 0
	FP := 0
	FN := 0
	elap := make([]int64, 0)
	nomalStatusCode, getNomalStatusCodeErr := getNomalStatusCode(target, mHost)
	blockStatusCode, getBlockStatusCodeerr := getNomalStatusCode(target+`/keys?1%20AND%201=1%20UNION%20ALL%20SELECT%201,NULL,%27<script>alert("XSS")</script>%27,table_name%20FROM%20information_schema.tables%20WHERE%202>1--/**/;%20EXEC%20xp_cmdshell(%27cat%20../../../etc/passwd%27)#`, mHost)
	if getNomalStatusCodeErr != nil || getBlockStatusCodeerr != nil {
		os.Exit(1)
	}
	if nomalStatusCode == blockStatusCode {
		fmt.Println("目标网站未开启waf")
		os.Exit(1)
	}

	var mutex sync.Mutex
	concurrency := thread
	wg := &sync.WaitGroup{}
	jobs := make(chan string, len(fileList))

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range jobs {
				isWhite, isPass, time, err := work(addr, isHttps, blockStatusCode, f)
				mutex.Lock()
				bar.Add(1)
				if err != nil {
					fmt.Printf("%s\n", err)
					continue
				}
				elap = append(elap, time)
				success++
				if isWhite {
					if isPass {
						TN += 1
					} else {
						FP += 1
					}
				} else {
					if isPass {
						FN += 1
					} else {
						TP += 1
					}
				}
				mutex.Unlock()
			}
		}()
	}

	for _, f := range fileList {
		jobs <- f
	}
	close(jobs)
	wg.Wait()

	fmt.Printf("总样本数量: %d    成功: %d    错误: %d\n", len(fileList), success, (len(fileList) - success))
	fmt.Printf("检出率: %.2f%% (恶意样本总数: %d , 正确拦截: %d , 漏报放行: %d)\n", float64(TP)*100/float64(TP+FN), TP+FN, TP, FN)
	fmt.Printf("误报率: %.2f%% (正常样本总数: %d , 正确放行: %d , 误报拦截: %d)\n", float64(FP)*100/float64(TN+FP), TN+FP, TN, FP)
	fmt.Printf("准确率: %.2f%% (正确拦截 + 正确放行）/样本总数 \n", float64(TP+TN)*100/float64(TP+TN+FP+FN))

	all := len(elap)
	sort.Slice(elap, func(i, j int) bool { return elap[i] < elap[j] })
	var sum int64 = 0
	for _, v := range elap {
		sum += v
	}
	fmt.Printf("平均耗时: %.2f 毫秒\n", float64(sum)/float64(all)/1000000)
}
