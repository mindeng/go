# 网络相关小工具

## dns_test.go

该文件测试了 golang 的域名解析功能，主要有如下几点：

1. 获取域名绑定的多个 IP 地址：`net.LookupIP()`
2. 指定某个特定 IP 向域名发起 HTTP 请求
   - 通过自定义 `http.Transport` 实现
   - 类似 `curl --resolve www.google.com:443:<ip> https://www.google.com`
3. 捕获 HTTP 请求的底层连接信息：通过 `httptrace.ClientTrace` 实现

## fastestip.go

找到访问某个域名最快的 IP 地址，具体可参考 `FastestAddress()` 函数，以及 `fastestip_test.go` 测试用例。
