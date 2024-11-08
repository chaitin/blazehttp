package main

import (
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/chaitin/blazehttp/testcases"
	"github.com/chaitin/blazehttp/utils"
	"github.com/chaitin/blazehttp/worker"
	"github.com/schollz/progressbar/v3"
)

const (
	NoneTag = "none" // http file without tag
)

var (
	target            string // the target web site, example: http://192.168.0.1:8080
	glob              string // use glob expression to select multi files
	timeout           = 1000 // default 1000 ms
	c                 = 10   // default 10 concurrent workers
	s                 = 0    // default 0 sleep second
	mHost             string // modify host header
	requestPerSession bool   // send request per session
)

func init() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./blazehttp -t <url>")
		os.Exit(1)
	}
	flag.StringVar(&target, "t", "", "target website, example: http://192.168.0.1:8080")
	flag.IntVar(&c, "c", 10, "concurrent workers, default 10")
	flag.IntVar(&s, "s", 0, "sleep second, default 0")
	flag.StringVar(&glob, "g", "", "glob expression, example: *.http")
	flag.IntVar(&timeout, "timeout", 1000, "connection timeout, default 1000 ms")
	flag.StringVar(&mHost, "H", "", "modify host header")
	flag.BoolVar(&requestPerSession, "rps", true, "send request per session")
	flag.Parse()
	if url, err := url.Parse(target); err != nil || url.Scheme == "" || url.Host == "" {
		fmt.Println("invalid target url, example: http://chaitin.com:9443")
		os.Exit(1)
	}
}

func main() {
	var addr string
	var isHttps bool

	if strings.HasPrefix(target, "http") {
		u, _ := url.Parse(target)
		if u.Scheme == "https" {
			isHttps = true
		}
		addr = u.Host
	}

	isWaf, blockStatusCode, err := utils.GetWafBlockStatusCode(target, mHost)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !isWaf {
		fmt.Println("目标网站未开启waf")
		os.Exit(1)
	}

	fileList := make([]string, 0)
	if glob == "" {
		if err := fs.WalkDir(testcases.EmbedTestCasesFS, ".", func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			fileList = append(fileList, path)
			return nil
		}); err != nil {
			panic(err)
		}
	} else {
		globFiles, err := utils.GetAllFiles(glob)
		if err != nil {
			fmt.Printf("open %s error: %s\n", glob, err)
			return
		}
		fileList = globFiles
	}

	if len(fileList) == 0 {
		fmt.Println("no test case found")
		return
	}
	// progress bar
	progressBar := progressbar.NewOptions64(
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

	worker := worker.NewWorker(
		addr,
		isHttps,
		fileList,
		blockStatusCode,
		worker.WithConcurrence(c),
		worker.WithSleep(s),
		worker.WithReqHost(mHost),
		worker.WithReqPerSession(requestPerSession),
		worker.WithTimeout(timeout),
		worker.WithUseEmbedFS(glob == ""), // use embed test case fs when glob is empty
		worker.WithProgressBar(progressBar),
	)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		worker.Stop()
	}()
	worker.Run()
}
