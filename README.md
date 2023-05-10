# BlazeHTTP

一个支持非标准HTTP请求/响应解析的库，附送一个发送大量非标准HTTP请求测试的工具。Enjoy yourself!

(Named by GPT!)

## 轮子?

公交车的轮子转啊转, 转啊转~, 跑题了~

该项目是为解决下面问题:

1. 标准库不支持解析`畸形的HTTP请求`
2. 请求文件需要有`标签`等标注信息
3. 没有免费的工具发送`大量`的`HTTP请求`
4. 怎么确定WAF工作了？附送一些攻击样本

> 如果项目对您有用, 欢迎star、fork!
> 如果项目有任何问题，欢迎提PR!

## 使用帮助

### 以库形式引用

```bash
go get github.com/chatin/blazehttp/http
```

### 命令行工具

```bash
go build ./cmd/blazehttp
```

## 小试牛刀

```bash
# 测试请求
./blazehttp -t http://192.168.0.1:8080 -g './testcases/*.http'

sending 100% |██████████████████████████████████████████| (18/18, 86 it/s)        
Total http file: 18, success: 18 failed: 0
Stat http response code

Status code: 403 hit: 16
Status code: 200 hit: 2

Stat http request tag

tag: sqli hit: 1
tag: black hit: 16
tag: file_include hit: 1
tag: file_upload hit: 1
tag: java_unserialize hit: 1
tag: php_unserialize hit: 1
tag: cmdi hit: 1
tag: ssrf hit: 1
tag: xslti hit: 1
tag: xss hit: 1
tag: xxe hit: 1
tag: asp_code hit: 1
tag: white hit: 2
tag: ognl hit: 1
tag: shellshock hit: 1
tag: ssti hit: 1
tag: directory_traversal hit: 1
tag: php_code hit: 1
```
