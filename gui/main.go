package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	fTheme "fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/chaitin/blazehttp/gui/theme"
	"github.com/chaitin/blazehttp/testcases"
	"github.com/chaitin/blazehttp/utils"
	"github.com/chaitin/blazehttp/worker"
	"github.com/samber/lo"
)

var (
	allTestData     []string
	rawTestCaseData = make([][]string, 0)
	testResultData  = make([][]string, 0)
)

func init() {
	rawTestCaseData = [][]string{
		{"文件名", "样本属性", "请求状态", "响应状态", "请求耗时(ms)", "是否拦截"},
	}
	testResultData = [][]string{
		{"文件名", "样本属性", "请求状态", "响应状态", "请求耗时(ms)", "是否拦截"},
	}

	if err := fs.WalkDir(testcases.EmbedTestCasesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		allTestData = append(allTestData, path)
		fileName := path
		fileProperty := "正常"
		if !strings.HasSuffix(path, "white") {
			fileProperty = "恶意"
		}

		rawTestCaseData = append(rawTestCaseData, []string{fileName, fileProperty, "-", "-", "-", "-"})
		return nil
	}); err != nil {
		panic(err)
	}
}

type Result struct {
	Total          binding.String
	NormalTotal    binding.String
	MaliciousTotal binding.String

	RequestTotal binding.String
	Success      binding.String
	Error        binding.String

	AvgTime binding.String
	P95Time binding.String
	P90Time binding.String

	DetectionRate  binding.String
	SuccessBlocked binding.String
	ErrorAllowed   binding.String

	FalsePositiveRate binding.String
	SuccessAllowed    binding.String
	ErrorBlocked      binding.String

	Accuracy binding.String
}

func NewResult() *Result {
	return &Result{
		Total:          binding.NewString(),
		NormalTotal:    binding.NewString(),
		MaliciousTotal: binding.NewString(),

		RequestTotal: binding.NewString(),
		Success:      binding.NewString(),
		Error:        binding.NewString(),

		DetectionRate:  binding.NewString(),
		SuccessBlocked: binding.NewString(),
		ErrorAllowed:   binding.NewString(),

		FalsePositiveRate: binding.NewString(),
		SuccessAllowed:    binding.NewString(),
		ErrorBlocked:      binding.NewString(),

		Accuracy: binding.NewString(),
		AvgTime:  binding.NewString(),
	}
}

func (r *Result) SyncResult(result *worker.Result) {
	_ = r.RequestTotal.Set(fmt.Sprintf("总请求: %d", result.Success+result.Error))
	_ = r.Success.Set(fmt.Sprintf("请求成功: %d", result.Success))
	_ = r.Error.Set(fmt.Sprintf("请求出错: %d", result.Error))

	_ = r.DetectionRate.Set(fmt.Sprintf("检出率: %.2f%%", float64(result.TP)*100/float64(result.TP+result.FN)))
	_ = r.SuccessBlocked.Set(fmt.Sprintf("正确拦截: %d", result.TP))
	_ = r.ErrorAllowed.Set(fmt.Sprintf("漏报放行: %d", result.FN))

	_ = r.FalsePositiveRate.Set(fmt.Sprintf("误报率: %.2f%%", float64(result.FP)*100/float64(result.TN+result.FP)))
	_ = r.SuccessAllowed.Set(fmt.Sprintf("正确放行: %d", result.TN))
	_ = r.ErrorBlocked.Set(fmt.Sprintf("误报拦截: %d", result.FP))

	_ = r.Accuracy.Set(fmt.Sprintf("准确率: %.2f%% (正确拦截+正确放行) / 总样本", float64(result.TP+result.TN)*100/float64(result.TP+result.TN+result.FP+result.FN)))

	_ = r.AvgTime.Set(fmt.Sprintf("平均耗时: %.2f毫秒", float64(result.SuccessTimeCost)/float64(result.Success)/1000000))

	if result.Job != nil {
		fileProperty := "正常"
		if !strings.HasSuffix(result.Job.FilePath, "white") {
			fileProperty = "恶意"
		}

		reqSuccess := "失败"
		reqTime := "-"
		reqPass := "否"

		if result.Job.Result.Success {
			reqSuccess = "成功"
			if !result.Job.Result.IsPass {
				reqPass = "是"
			}
			reqTime = fmt.Sprintf("%d ms", result.Job.Result.TimeCost/1000000)
		}
		testResultData = append(testResultData, []string{result.Job.FilePath, fileProperty, reqSuccess, fmt.Sprintf("%d", result.Job.Result.StatusCode), reqTime, reqPass})
	}
}

func main() {
	a := app.New()
	a.Settings().SetTheme(&theme.BlazeHTTPTheme{})
	w := a.NewWindow("BLAZEHTTP")
	w.Resize(fyne.Size{Width: 810})

	outputCh := make(chan string)
	resultCh := make(chan *worker.Result)

	// sync result
	r := NewResult()
	initResult := func() {
		m := lo.CountValuesBy(rawTestCaseData, func(item []string) string {
			if item[1] == "正常" {
				return "正常"
			} else if item[1] == "恶意" {
				return "恶意"
			}
			return "unknown"
		})

		_ = r.Total.Set(fmt.Sprintf("总样本: %d", len(allTestData)))
		_ = r.NormalTotal.Set(fmt.Sprintf("正常样本: %d", m["正常"]))
		_ = r.MaliciousTotal.Set(fmt.Sprintf("恶意样本: %d", m["恶意"]))
		r.SyncResult(&worker.Result{})
	}
	initResult()
	go func() {
		for {
			select {
			case o := <-outputCh:
				d := dialog.NewError(errors.New(o), w)
				d.Show()
			case result := <-resultCh:
				r.SyncResult(result)
			}
		}
	}()

	// create tabs
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("", fTheme.ComputerIcon(), container.NewVBox(
			MakeRunForm(w, outputCh, resultCh, initResult),
			MakeRunResult(w, r),
		)),
		container.NewTabItemWithIcon("", fTheme.DocumentIcon(),
			MakeTestCaseTab(w),
		),
	)
	tabs.SetTabLocation(container.TabLocationLeading)

	// show window
	w.SetContent(
		tabs,
	)
	w.ShowAndRun()
}

func MakeRunResult(w fyne.Window, r *Result) fyne.CanvasObject {
	result0 := container.NewGridWithColumns(3,
		widget.NewLabelWithData(r.Total),
		widget.NewLabelWithData(r.NormalTotal),
		widget.NewLabelWithData(r.MaliciousTotal),

		widget.NewLabelWithData(r.RequestTotal),
		widget.NewLabelWithData(r.Success),
		widget.NewLabelWithData(r.Error),
	)

	result1 := container.NewGridWithColumns(3,
		widget.NewLabelWithData(r.DetectionRate),
		widget.NewLabelWithData(r.SuccessBlocked),
		widget.NewLabelWithData(r.ErrorAllowed),
	)
	result2 := container.NewGridWithColumns(3,
		widget.NewLabelWithData(r.FalsePositiveRate),
		widget.NewLabelWithData(r.SuccessAllowed),
		widget.NewLabelWithData(r.ErrorBlocked),
	)

	result3 := container.NewGridWithColumns(1,
		widget.NewLabelWithData(r.Accuracy),
	)

	result4 := container.NewGridWithColumns(1,
		widget.NewLabelWithData(r.AvgTime),
	)
	return container.NewVBox(result0, result1, result2, result3, result4)
}

func MakeRunForm(w fyne.Window, outputCh chan string, resultCh chan *worker.Result, initResult func()) fyne.CanvasObject {
	// website url
	target := widget.NewEntry()
	target.SetText("https://demo.waf-ce.chaitin.cn")
	target.Validator = validation.NewRegexp(`^https?://`, "必须以http或https开头")
	// concurrent workers
	workers := widget.NewEntry()
	workers.SetText("10")
	workers.Validator = validation.NewRegexp(`^\d+$`, "工作线程必须是数字")

	sleep := widget.NewEntry()
	sleep.SetText("0")
	sleep.Validator = validation.NewRegexp(`^\d+$`, "请求间隔必须是数字")
	
	// modify host header
	reqHost := widget.NewEntry()
	// timeout
	timeout := widget.NewEntry()
	timeout.SetText("1000")
	timeout.Validator = validation.NewRegexp(`^\d+$`, "请求超时必须是数字")

	// timeout
	statusCode := widget.NewEntry()
	statusCode.SetText("403")
	statusCode.Validator = validation.NewRegexp(`^\d+$`, "StatusCode必须是数字")

	advanceForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "修改请求(Host)", Widget: reqHost, HintText: ""},
			{Text: "请求超时", Widget: timeout, HintText: ""},
			{Text: "拦截状态码(respCode)", Widget: statusCode, HintText: "一般情况下WAF拦截状态码为403"},
		},
	}

	advanceDialog := dialog.NewCustomConfirm("高级配置", "确认", "取消", container.NewVBox(advanceForm), nil, w)
	advanceDialog.Resize(fyne.NewSize(500, 0))
	advanceBtn := widget.NewButton("其他配置", func() {
		advanceDialog.Show()
	})

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "目标网站", Widget: target, HintText: ""},
			{Text: "工作线程", Widget: workers, HintText: ""},
			{Text: "请求间隔（秒）", Widget: sleep, HintText: "注：请求间隔用于外部waf频率过高会封ip时候选择，使用时结合工作线程计算频率"},
		},
	}
	// cancel
	cancelBtn := widget.NewButton("取消", func() {})
	cancelBtn.Hidden = true
	runBtn := widget.NewButton("运行", func() {})
	runBtn.Importance = widget.HighImportance
	stopCh := make(chan struct{})

	cancelBtn.OnTapped = func() {
		cancelBtn.Hidden = true
		runBtn.Hidden = false
		stopCh <- struct{}{}
	}

	runBtn.OnTapped = func() {
		if err := form.Validate(); err != nil {
			outputCh <- err.Error()
			return
		}
		if err := advanceForm.Validate(); err != nil {
			outputCh <- err.Error()
			return
		}
		runBtn.Hidden = true
		cancelBtn.Hidden = false
		initResult()
		testResultData = testResultData[:1]
		go func() {
			workersText := strings.TrimSpace(workers.Text)
			worksNum, _ := strconv.Atoi(workersText)

			sleepText := strings.TrimSpace(sleep.Text)
			sleepNum, _ := strconv.Atoi(sleepText)

			statusCode := strings.TrimSpace(statusCode.Text)
			statusCodeI, _ := strconv.Atoi(statusCode)

			err := run(target.Text, reqHost.Text, worksNum,sleepNum, statusCodeI, resultCh, stopCh)
			if err != nil {
				outputCh <- err.Error()
			}
			runBtn.Hidden = false
			cancelBtn.Hidden = true
		}()
	}
	return container.NewVBox(form, advanceBtn, runBtn, cancelBtn)
}

func MakeTestCaseTab(w fyne.Window) fyne.CanvasObject {
	tableData := rawTestCaseData

	fileProperty := widget.NewCheckGroup([]string{"正常", "恶意"}, nil)
	fileProperty.Selected = []string{"正常", "恶意"}
	fileProperty.Horizontal = true

	isIntercept := widget.NewCheckGroup([]string{"是", "否"}, nil)
	isIntercept.Selected = []string{"是", "否"}
	isIntercept.Horizontal = true

	result := widget.NewRadioGroup([]string{"原始样本", "测试结果"}, nil)
	result.SetSelected("原始样本")
	result.Horizontal = true

	tableFilterForm := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "列表数据", Widget: result, HintText: ""},
			{Text: "样本属性", Widget: fileProperty, HintText: ""},
			{Text: "是否拦截", Widget: isIntercept, HintText: ""},
		},
	}

	table := widget.NewTable(
		func() (int, int) {
			return len(tableData), len(tableData[0])
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			o.(*widget.Label).SetText(tableData[i.Row][i.Col])
		})
	table.SetColumnWidth(0, 300)
	table.SetColumnWidth(1, 70)
	table.SetColumnWidth(2, 70)
	table.SetColumnWidth(3, 70)
	table.SetColumnWidth(4, 100)
	table.OnSelected = func(id widget.TableCellID) {
		if id.Col == 0 && id.Row > 0 {
			f, err := testcases.EmbedTestCasesFS.ReadFile(tableData[id.Row][id.Col])
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			content := widget.NewMultiLineEntry()
			content.SetText(string(f))
			d := dialog.NewCustom("文件内容", "ok", content, w)
			d.Resize(fyne.NewSize(500, 400))
			d.Show()
		}
	}

	refreshTable := func(selected string, fileProperty, isIntercept []string) {
		var tempData [][]string
		if selected == "原始样本" {
			tempData = rawTestCaseData
		} else {
			tempData = testResultData
		}

		tableData = lo.Filter(tempData, func(row []string, i int) bool {
			if i == 0 {
				return true
			}
			if len(fileProperty) == 0 && len(isIntercept) == 0 {
				return false
			}
			if len(fileProperty) == 0 {
				for _, s := range isIntercept {
					if row[5] == s || row[5] == "-" {
						return true
					}
				}
				return false
			}
			if len(isIntercept) == 0 {
				for _, p := range fileProperty {
					if row[1] == p {
						return true
					}
				}
				return false
			}

			for _, p := range fileProperty {
				for _, s := range isIntercept {
					if row[1] == p && (row[5] == s || row[5] == "-") {
						return true
					}
				}
			}
			return false
		})
		table.ScrollTo(widget.TableCellID{Row: 0, Col: 0})
		table.Refresh()
	}

	fileProperty.OnChanged = func(selected []string) {
		refreshTable(result.Selected, selected, isIntercept.Selected)
	}
	isIntercept.OnChanged = func(selected []string) {
		refreshTable(result.Selected, fileProperty.Selected, selected)
	}
	result.OnChanged = func(selected string) {
		refreshTable(selected, fileProperty.Selected, isIntercept.Selected)
	}

	exportBtn := widget.NewButton("导出", nil)

	exportBtn.OnTapped = func() {
		dialog.ShowFileSave(func(writer fyne.URIWriteCloser, err error) {
			if writer == nil {
				return
			}
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			defer writer.Close()
			b := new(bytes.Buffer)
			csvWriter := csv.NewWriter(b)
			if err := csvWriter.WriteAll(tableData); err != nil {
				dialog.ShowError(err, w)
				return
			}
			if err := csvWriter.Error(); err != nil {
				dialog.ShowError(err, w)
				return
			}
			n, err := writer.Write(b.Bytes())
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			fmt.Println("Wrote", n, "bytes")
		}, w)
	}

	return container.NewBorder(tableFilterForm, nil, nil, exportBtn, table)
}

func run(target, mHost string, c,sleepNum, statusCode int, resultCh chan *worker.Result, stopCh chan struct{}) error {
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
		return err
	}
	if !isWaf {
		return errors.New("目标网站未开启waf")
	}
	if blockStatusCode != statusCode {
		return fmt.Errorf("探测到拦截状态码: %d 与配置拦截状态码: %d 不一致", blockStatusCode, statusCode)
	}

	worker := worker.NewWorker(
		addr,
		isHttps,
		allTestData,
		blockStatusCode,
		worker.WithConcurrence(c),
		worker.WithSleep(sleepNum),
		worker.WithReqHost(mHost),
		worker.WithUseEmbedFS(true), // use embed test case fs when glob is empty
		worker.WithResultCh(resultCh),
	)
	go func() {
		<-stopCh
		worker.Stop()
	}()
	worker.Run()

	return nil
}
