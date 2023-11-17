# BlazeHTTP

一个可以帮您测试 WAF 关键指标的工具

主要为了解决下面问题:

1. 标准库不支持解析`畸形的HTTP请求`
2. 没有免费的工具发送`大量`的`HTTP请求`
3. 没有免费的工具可以测试 WAF 的关键指标

> 如果项目对您有用, 欢迎star、fork!
> 如果项目有任何问题，欢迎提PR!

## 使用帮助

### 编译 or 下载 release

```bash
go build -o ./build/blazehttp ./cmd/blazehttp
```

### 开始测试

```bash
./build/blazehttp http://127.0.0.1:8008
sending 100% |██████████████████████████████████████████| (33669/33669, 943 it/s) [35s:0s]
TP[攻击拦截]: 412    TN[正常放行]: 33071    FP[误报]: 23    FN[漏报]: 163
总样本数量: 33669    成功: 33669    错误: 0
检出率: 71.65%
误报率: 5.29%
准确率: 99.45%

90% 平均耗时: 0.67毫秒
99% 平均耗时: 0.87毫秒
平均耗时: 0.87毫秒
```

### 环境准备（推荐）

nginx.conf

``` conf
location / {
    return 200 'hello WAF!';
    default_type text/plain;
}
```
启动 web 服务，并接入 waf
``` bash
docker run -d -p 8088:80 -v /path/to/nginx.conf:/etc/nginx/nginx.conf -d nginx:latest
```