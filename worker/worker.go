package worker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	blazehttp "github.com/chaitin/blazehttp/http"

	"github.com/chaitin/blazehttp/testcases"
)

type Progress interface {
	Add(n int) error
}

type TestResult struct {
	FilePath        string
	IsBlocked       bool
	StatusCode      int
	Error           error
	ResponseDetails string // 根据需要添加更多字段
}

type Worker struct {
	ctx    context.Context
	cancel context.CancelFunc

	concurrence   int // concurrent connections
	fileList      []string
	jobs          chan *Job
	jobResult     chan *Job
	jobResultDone chan struct{}
	result        *Result
	progressBar   Progress
	results       chan TestResult

	addr            string // target addr
	isHttps         bool   // is https
	timeout         int    // connection timeout
	blockStatusCode int    // block status code
	reqHost         string // request host of header
	reqPerSession   bool   // request per session
	useEmbedFS      bool
	resultCh        chan *Result
}

type WorkerOption func(*Worker)

func WithTimeout(timeout int) WorkerOption {
	return func(w *Worker) {
		w.timeout = timeout
	}
}

func WithReqHost(reqHost string) WorkerOption {
	return func(w *Worker) {
		w.reqHost = reqHost
	}
}

func WithReqPerSession(reqPerSession bool) WorkerOption {
	return func(w *Worker) {
		w.reqPerSession = reqPerSession
	}
}

func WithUseEmbedFS(useEmbedFS bool) WorkerOption {
	return func(w *Worker) {
		w.useEmbedFS = useEmbedFS
	}
}

func WithConcurrence(c int) WorkerOption {
	return func(w *Worker) {
		w.concurrence = c
	}
}

func WithResultCh(ch chan *Result) WorkerOption {
	return func(w *Worker) {
		w.resultCh = ch
	}
}

func WithProgressBar(pb Progress) WorkerOption {
	return func(w *Worker) {
		w.progressBar = pb
	}
}

func (w *Worker) Stop() {
	w.cancel()
}

func NewWorker(
	addr string,
	isHttps bool,
	fileList []string,
	blockStatusCode int,
	options ...WorkerOption,
) *Worker {
	w := &Worker{
		concurrence: 10, // default 10

		// payloads
		fileList: fileList,

		// connect target & config
		addr:            addr,
		isHttps:         isHttps,
		timeout:         1000, // 1000ms
		results:         make(chan TestResult, len(fileList)),
		blockStatusCode: blockStatusCode,

		jobs:          make(chan *Job),
		jobResult:     make(chan *Job),
		jobResultDone: make(chan struct{}),

		result: &Result{
			Total: int64(len(fileList)),
		},
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())

	for _, opt := range options {
		opt(w)
	}
	return w
}

type Job struct {
	FilePath string
	Result   *JobResult
}

type JobResult struct {
	IsWhite    bool
	IsPass     bool
	Success    bool
	TimeCost   int64
	StatusCode int
	Err        string
}

type Result struct {
	Total           int64 // total poc
	Error           int64
	Success         int64 // success poc
	SuccessTimeCost int64 // success success cost
	TN              int64
	FN              int64
	TP              int64
	FP              int64
	Job             *Job
}

type Output struct {
	Out string
	Err string
}

func (w *Worker) Run() {
	go func() {
		w.jobProducer()
	}()

	go func() {
		w.processJobResult()
		w.jobResultDone <- struct{}{}
	}()

	wg := sync.WaitGroup{}

	for i := 0; i < w.concurrence; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.runWorker()
		}()
	}
	wg.Wait()

	close(w.jobResult)
	<-w.jobResultDone

	fmt.Println(w.generateResult())
}

func (w *Worker) runWorker() {
	for job := range w.jobs {
		func() {
			defer func() {
				w.jobResult <- job
			}()
			filePath := job.FilePath
			req := new(blazehttp.Request)
			if w.useEmbedFS {
				if err := req.ReadFileFromFS(testcases.EmbedTestCasesFS, filePath); err != nil {
					job.Result.Err = fmt.Sprintf("read request file: %s from embed fs error: %s\n", filePath, err)
					return
				}
			} else {
				if err := req.ReadFile(filePath); err != nil {
					job.Result.Err = fmt.Sprintf("read request file: %s error: %s\n", filePath, err)
					return
				}
			}

			if w.reqHost != "" {
				req.SetHost(w.reqHost)
			} else {
				req.SetHost(w.addr)
			}

			if w.reqPerSession {
				// one http request one connection
				req.SetHeader("Connection", "close")
			}

			req.CalculateContentLength()

			start := time.Now()
			conn := blazehttp.Connect(w.addr, w.isHttps, w.timeout)
			if conn == nil {
				job.Result.Err = fmt.Sprintf("connect to %s failed!\n", w.addr)
				return
			}
			nWrite, err := req.WriteTo(*conn)
			if err != nil {
				job.Result.Err = fmt.Sprintf("send request poc: %s length: %d error: %s", filePath, nWrite, err)
				return
			}

			rsp := new(blazehttp.Response)
			if err = rsp.ReadConn(*conn); err != nil {
				job.Result.Err = fmt.Sprintf("read poc file: %s response, error: %s", filePath, err)
				return
			}
			elap := time.Since(start).Nanoseconds()
			err = (*conn).Close()
			if err != nil {
				return
			}
			job.Result.Success = true
			if strings.HasSuffix(job.FilePath, "white") {
				job.Result.IsWhite = true // white case
			}

			code := rsp.GetStatusCode()
			job.Result.StatusCode = code
			if code != w.blockStatusCode {
				job.Result.IsPass = true
			}
			job.Result.TimeCost = elap

			// 生成 TestResult 并发送到 results 通道
			testResult := TestResult{
				FilePath:   filePath,
				IsBlocked:  code == w.blockStatusCode,
				StatusCode: code,
				Error:      nil,
			}
			if job.Result.Err != "" {
				testResult.Error = fmt.Errorf(job.Result.Err)
			}
			w.results <- testResult
		}()
	}
}

func parseError(errStr string) error {
	if errStr == "" {
		return nil
	}
	return fmt.Errorf(errStr)
}

func (w *Worker) processJobResult() {
	// 定义两个切片用于存储误报和漏报的测试用例
	var falsePositives []TestResult
	var falseNegatives []TestResult

	for job := range w.jobResult {
		if job.Result.Success {
			w.result.Success++
			w.result.SuccessTimeCost += job.Result.TimeCost
			if job.Result.IsWhite {
				if job.Result.IsPass {
					w.result.TN++
				} else {
					w.result.FP++
					// 记录误报的测试用例
					falsePositives = append(falsePositives, TestResult{
						FilePath:   job.FilePath,
						IsBlocked:  true,
						StatusCode: job.Result.StatusCode,
						Error:      parseError(job.Result.Err),
					})
				}
			} else {
				if job.Result.IsPass {
					w.result.FN++
					// 记录漏报的测试用例
					falseNegatives = append(falseNegatives, TestResult{
						FilePath:   job.FilePath,
						IsBlocked:  false,
						StatusCode: job.Result.StatusCode,
						Error:      parseError(job.Result.Err),
					})
				} else {
					w.result.TP++
				}
			}
		} else {
			w.result.Error++
		}
		if w.resultCh != nil {
			r := *w.result
			r.Job = job
			w.resultCh <- &r
		}
	}

	// 在所有结果处理完毕后，写入误报和漏报的测试用例到CSV文件
	w.writeResults("false_positives.csv", falsePositives)
	w.writeResults("false_negatives.csv", falseNegatives)
}

func (w *Worker) jobProducer() {
	defer close(w.jobs)
	for _, f := range w.fileList {
		select {
		case <-w.ctx.Done():
			return
		default:
			w.jobs <- &Job{
				FilePath: f,
				Result:   &JobResult{},
			}
			if w.progressBar != nil {
				_ = w.progressBar.Add(1)
			}
		}
	}
}

func (w *Worker) generateResult() string {
	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("总样本数量: %d    成功: %d    错误: %d\n", w.result.Total, w.result.Success, w.result.Total-w.result.Success))
	sb.WriteString(fmt.Sprintf("检出率: %.2f%% (恶意样本总数: %d , 正确拦截: %d , 漏报放行: %d)\n", float64(w.result.TP)*100/float64(w.result.TP+w.result.FN), w.result.TP+w.result.FN, w.result.TP, w.result.FN))
	sb.WriteString(fmt.Sprintf("误报率: %.2f%% (正常样本总数: %d , 正确放行: %d , 误报拦截: %d)\n", float64(w.result.FP)*100/float64(w.result.TN+w.result.FP), w.result.TN+w.result.FP, w.result.TN, w.result.FP))
	sb.WriteString(fmt.Sprintf("准确率: %.2f%% (正确拦截 + 正确放行）/样本总数 \n", float64(w.result.TP+w.result.TN)*100/float64(w.result.TP+w.result.TN+w.result.FP+w.result.FN)))
	sb.WriteString(fmt.Sprintf("平均耗时: %.2f毫秒\n", float64(w.result.SuccessTimeCost)/float64(w.result.Success)/1000000))
	return sb.String()
}

// 新增的写入CSV文件的函数
func (w *Worker) writeResults(filename string, results []TestResult) {
	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("无法创建 %s: %v\n", filename, err)
		return
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)

	// 写入 CSV 头部
	_, err = f.WriteString("FilePath,IsBlocked,StatusCode,Error\n")
	if err != nil {
		return
	}

	for _, res := range results {
		errorMsg := ""
		if res.Error != nil {
			errorMsg = res.Error.Error()
		}
		line := fmt.Sprintf("\"%s\",%t,%d,\"%s\"\n", res.FilePath, res.IsBlocked, res.StatusCode, errorMsg)
		_, err := f.WriteString(line)
		if err != nil {
			return
		}
	}
}
