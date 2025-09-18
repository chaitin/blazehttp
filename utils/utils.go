package utils

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func getNormalStatusCode(url string, mHost string) (statusCode int, conErr error) {
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
		return -1, fmt.Errorf("创建请求失败: %s", err)
	}
	if mHost != "" {
		req.Host = mHost
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/99.0.9999.999 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := client.Do(req)
	if err != nil {
		return -1, fmt.Errorf("HTTP 请求发生错误: %s", err)
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode
	return
}

func GetWafBlockStatusCode(target, mHost string) (isWaf bool, statusCode int, err error) {
	var normalStatusCode, blockStatusCode int
	normalStatusCode, err = getNormalStatusCode(target+`/abcdefg/hijklmn/a.html`, mHost)
	if err != nil {
		return
	}
	blockStatusCode, err = getNormalStatusCode(target+`/abcdefg/hijklmn/a.html?1%20AND%201=1%20UNION%20ALL%20SELECT%201,NULL,%27<script>alert("XSS")</script>%27,table_name%20FROM%20information_schema.tables%20WHERE%202>1--/**/;%20EXEC%20xp_cmdshell(%27cat%20../../../etc/passwd%27)#`, mHost)
	if err != nil {
		return
	}
	if normalStatusCode != blockStatusCode {
		isWaf = true
	}
	statusCode = blockStatusCode
	return
}

func GetAllFiles(glob string) ([]string, error) {
	var files []string
	matches, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	for _, filePath := range matches {
		info, err := os.Stat(filePath)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			files = append(files, filePath)
		}
	}
	return files, nil
}
