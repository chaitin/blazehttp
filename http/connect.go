package http

import (
	"bufio"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"time"

	"golang.org/x/net/proxy"
)

func dialProxy(addr, proxyString string, dialer *net.Dialer) (conn net.Conn, err error) {

	proxyUrl, err := url.Parse(proxyString)
	if err != nil {
		return nil, err
	}
	if proxyUrl.Scheme == "socks5" {
		var auth *proxy.Auth = nil;
		if proxyUrl.User != nil {
			username := proxyUrl.User.Username()
			password, _ := proxyUrl.User.Password()
			auth = &proxy.Auth{
				User: username,
				Password: password,
			}
		}
		dialer, err := proxy.SOCKS5("tcp", proxyUrl.Host, auth, dialer)
		if err != nil {
			return nil, err
		}
		conn, err = dialer.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
	} else if proxyUrl.Scheme == "http" {
		conn, err = dialer.Dial("tcp", proxyUrl.Host)
		if err != nil {
			return nil, err
		}
		request := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", addr, addr)
		if proxyUrl.User != nil {
			username := proxyUrl.User.Username()
			password, _ := proxyUrl.User.Password()
			auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			request += "Proxy-Authorization: Basic " + auth + "\r\n"
		}
		request += "\r\n"
		_, err = fmt.Fprint(conn, request)
		if err != nil {
			conn.Close()
			return nil, err
		}

		reader := bufio.NewReader(conn)

		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if m, err := regexp.MatchString("HTTP/[\\d\\.]+ 200 Connection established", line); !m {
			conn.Close()
			return nil, fmt.Errorf("CONNECT请求失败: %s %s", line, err)
		}

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				conn.Close()
				return nil, fmt.Errorf("读取CONNECT响应头失败: %v", err)
			}
			if line == "\r\n" {
				break
			}
		}
	} else {
		return nil, fmt.Errorf("不支持的代理类型: " + proxyUrl.Scheme)
	}
	if err != nil {
		return nil, fmt.Errorf("连接失败: %v", err)
	}
	return conn, nil
}

func Connect(addr string, proxyString string, isHttps bool, timeout int) (conn net.Conn, err error) {
	if m, _ := regexp.MatchString(`.*(]:)|(:)[0-9]+$`, addr); !m {
		if isHttps {
			addr = fmt.Sprintf("%s:443", addr)
		} else {
			addr = fmt.Sprintf("%s:80", addr)
		}
	}
	retryCnt := 0
retry:
	dialer := net.Dialer{
		Timeout: time.Duration(timeout) * time.Millisecond * 2,
	}
	if proxyString != "" {
		conn, err = dialProxy(addr, proxyString, &dialer)
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		retryCnt++
		if retryCnt < 4 {
			goto retry
		} else {
			return nil, err
		}
	}
	if isHttps {
		tlsConn := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		if err := tlsConn.Handshake(); err != nil {
			retryCnt++
			if retryCnt < 4 {
				goto retry
			} else {
				return nil, err
			}
		}
		conn = tlsConn
	}
	wDeadline := time.Now().Add(time.Duration(timeout) * time.Millisecond)
	rDeadline := time.Now().Add(time.Duration(timeout*2) * time.Millisecond)
	deadline := time.Now().Add(time.Duration(timeout*2) * time.Millisecond)
	_ = conn.SetDeadline(deadline)
	_ = conn.SetReadDeadline(rDeadline)
	_ = conn.SetWriteDeadline(wDeadline)

	return conn, nil
}
