package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const DefaultMaxBodySize int64 = 1024 * 2048 // 2MB

var (
	// This is what relay.plugin.Plugins will load to handle traffic plugin duties
	Plugin relayPlugin = New()

	hasPort                       = regexp.MustCompile(`:\d+$`)
	logger                        = log.New(os.Stdout, "[traffic-relay] ", 0)
	trafficRelayTargetVar         = "TRAFFIC_RELAY_TARGET"
	trafficRelayCookiesVar        = "TRAFFIC_RELAY_COOKIES"
	trafficRelayMaxBodySizeVar    = "TRAFFIC_RELAY_MAX_BODY_SIZE"
	trafficRelayOriginOverrideVar = "TRAFFIC_RELAY_ORIGIN_OVERRIDE"
)

type relayPlugin struct {
	transport      *http.Transport
	targetScheme   string          // http|https
	targetHost     string          // e.g. 192.168.0.1:1234
	originOverride string          // default to passing Origin header as-is, but set to override
	relayedCookies map[string]bool // the name of cookies that should be relayed
	maxBodySize    int64           // maximum length in bytes of relayed bodies
}

func New() relayPlugin {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
		Proxy:           http.ProxyFromEnvironment,
		IdleConnTimeout: 2 * time.Second, // TODO set from configs
	}
	return relayPlugin{
		transport,
		"",
		"",
		"",
		map[string]bool{},
		DefaultMaxBodySize,
	}
}

func (plug relayPlugin) Name() string {
	return "Relay"
}

func (plug relayPlugin) HandleRequest(clientResponse http.ResponseWriter, clientRequest *http.Request, serviced bool) bool {
	if serviced {
		return false
	}
	if plug.targetScheme == "" || plug.targetHost == "" {
		return false
	}
	if clientRequest.Header.Get("Upgrade") == "websocket" {
		return plug.handleUpgrade(clientResponse, clientRequest)
	} else {
		return plug.handleHttp(clientResponse, clientRequest)
	}
}

func (plug relayPlugin) ConfigVars() map[string]bool {
	return map[string]bool{
		trafficRelayTargetVar:         true,
		trafficRelayCookiesVar:        false,
		trafficRelayMaxBodySizeVar:    false,
		trafficRelayOriginOverrideVar: false,
	}
}

func (plug *relayPlugin) Config() bool {
	targetVar := os.Getenv(trafficRelayTargetVar)
	targetURL, err := url.Parse(targetVar)
	if err != nil {
		logger.Printf("Could not parse %v environment variable URL: %v", trafficRelayTargetVar, targetVar)
		return false
	}
	plug.targetScheme = targetURL.Scheme
	plug.targetHost = targetURL.Host

	originOverrideVar := os.Getenv(trafficRelayOriginOverrideVar)
	if len(originOverrideVar) > 0 {
		plug.originOverride = originOverrideVar
	}

	cookiesVar := os.Getenv(trafficRelayCookiesVar)
	if len(cookiesVar) > 0 {
		for _, cookieName := range strings.Split(cookiesVar, " ") { // Should we support spaces?
			plug.relayedCookies[cookieName] = true
		}
	}

	maxBodySizeVar := os.Getenv(trafficRelayMaxBodySizeVar)
	if len(maxBodySizeVar) > 0 {
		maxBodySize, err := strconv.ParseInt(maxBodySizeVar, 10, 64)
		if err != nil {
			logger.Println("Error parsing TRAFFIC_RELAY_MAX_BODY_SIZE environment variable", err)
			return false
		}
		plug.maxBodySize = maxBodySize
	}
	return true
}

func (plug *relayPlugin) prepRelayRequest(clientRequest *http.Request) {
	clientRequest.URL.Scheme = plug.targetScheme
	clientRequest.URL.Host = plug.targetHost
	clientRequest.Host = plug.targetHost
	if len(plug.originOverride) > 0 {
		clientRequest.Header.Set(
			"Origin",
			fmt.Sprintf("%v://%v", plug.targetScheme, plug.originOverride),
		)
	}
	if len(plug.relayedCookies) == 0 {
		clientRequest.Header.Del("Cookie")
	} else {
		var cookieString strings.Builder
		first := true
		for _, cookie := range clientRequest.Cookies() {
			if _, ok := plug.relayedCookies[cookie.Name]; ok == false {
				continue
			}
			if first == false {
				cookieString.WriteString("&")
			}
			cookieString.WriteString(cookie.String())
			first = false
		}
		clientRequest.Header.Set("Cookie", cookieString.String())
	}
}

func (plug *relayPlugin) handleHttp(clientResponse http.ResponseWriter, clientRequest *http.Request) bool {
	plug.prepRelayRequest(clientRequest)
	if !clientRequest.URL.IsAbs() {
		http.Error(clientResponse, fmt.Sprintf("This plugin can not respond to relative (non-absolute) requests: %v", clientRequest.URL), 500)
		return true
	}

	targetResponse, err := plug.transport.RoundTrip(clientRequest)
	if err != nil {
		logger.Printf("Cannot read response from server %v", err)
		return false
	}
	defer targetResponse.Body.Close()

	// Set the relayed headers
	for key, values := range targetResponse.Header {
		for _, value := range values {
			clientResponse.Header().Add(key, value)
		}
	}

	if targetResponse.ContentLength > plug.maxBodySize {
		clientResponse.WriteHeader(http.StatusServiceUnavailable)
		clientResponse.Write([]byte("Response body content-length was too large"))
	} else if targetResponse.ContentLength > 0 {
		clientResponse.WriteHeader(targetResponse.StatusCode)
		if _, err := io.CopyN(clientResponse, targetResponse.Body, targetResponse.ContentLength); err != nil {
			logger.Printf("Error relaying response body to client: %s", err)
		}
	} else if targetResponse.ContentLength < 0 {
		clientResponse.WriteHeader(targetResponse.StatusCode)
		if _, err := io.CopyN(clientResponse, targetResponse.Body, plug.maxBodySize); err != nil {
			logger.Printf("Error relaying response body with unknown content-length: %s", err)
		}
	} else {
		clientResponse.WriteHeader(targetResponse.StatusCode)
	}
	return true
}

func (plug *relayPlugin) handleUpgrade(clientResponse http.ResponseWriter, clientRequest *http.Request) bool {
	plug.prepRelayRequest(clientRequest)
	if !clientRequest.URL.IsAbs() {
		logger.Println("Url was not absolute", clientRequest.URL.Host)
		http.Error(clientResponse, fmt.Sprintf("This plugin can not respond to relative (non-absolute) requests: %v", clientRequest.URL), 500)
		return true
	}
	logger.Println("Upgrading to websocket:", clientRequest.URL)

	// Connect to the target WS service
	var targetConn net.Conn
	var err error
	if clientRequest.URL.Scheme == "https" {
		targetConn, err = tls.Dial("tcp", clientRequest.URL.Host, &tls.Config{})
		if err != nil {
			logger.Println("Error setting up target tls websocket", err)
			http.Error(clientResponse, fmt.Sprintf("Could not dial connect %v: %v", clientRequest.URL.Host, err), 404)
			return true
		}
	} else {
		targetConn, err = net.Dial("tcp", clientRequest.URL.Host)
		if err != nil {
			logger.Println("Error setting up target websocket", err)
			http.Error(clientResponse, fmt.Sprintf("Could not dial connect %v: %v", clientRequest.URL.Host, err), 404)
			return true
		}
	}

	// Write the original client request to the target
	requestLine := fmt.Sprintf("%v %v %v\r\nHost: %v\r\n", clientRequest.Method, clientRequest.URL.String(), clientRequest.Proto, clientRequest.Host)
	if _, err := io.WriteString(targetConn, requestLine); err != nil {
		logger.Printf("Could not write the WS request: %v", err)
		http.Error(clientResponse, fmt.Sprintf("Could not write the WS request: %v %v", clientRequest.URL.Host, err), 500)
		return true
	}
	headerBuffer := new(bytes.Buffer)
	if err := clientRequest.Header.Write(headerBuffer); err != nil {
		logger.Println("Could not write WS header to buffer", err)
		http.Error(clientResponse, fmt.Sprintf("Could not write the WS header: %v %v", clientRequest.URL.Host, err), 500)
		return true
	}
	_, err = headerBuffer.WriteTo(targetConn)
	if err != nil {
		logger.Println("Could not write WS header to target", err)
		http.Error(clientResponse, fmt.Sprintf("Could not write the final header line: %v %v", clientRequest.URL.Host, err), 500)
		return true
	}
	_, err = io.WriteString(targetConn, "\r\n")
	if err != nil {
		logger.Println("Could not complete WS header", err)
		http.Error(clientResponse, fmt.Sprintf("Could not write the final header line: %v %v", clientRequest.URL.Host, err), 500)
		return true
	}

	hij, ok := clientResponse.(http.Hijacker)
	if !ok {
		logger.Println("httpserver does not support hijacking")
		http.Error(clientResponse, "Does not support hijacking", 500)
		return true
	}

	clientConn, _, err := hij.Hijack()
	if err != nil {
		logger.Println("Cannot hijack connection ", err)
		http.Error(clientResponse, "Could not hijack", 500)
		return true
	}

	// And then relay everything between the client and target
	go transfer(targetConn, clientConn)
	transfer(clientConn, targetConn)
	return true
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()
	io.Copy(destination, source)
}

/*
Copyright 2019 FullStory, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy of this software
and associated documentation files (the "Software"), to deal in the Software without restriction,
including without limitation the rights to use, copy, modify, merge, publish, distribute,
sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all copies or
substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT
NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY,
WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
