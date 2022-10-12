package traffic

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fullstorydev/relay-core/relay/version"
)

const RelayVersionHeaderName = "X-Relay-Version"

var logger = log.New(os.Stdout, "[relay-traffic] ", 0)

// Handler handles HTTP traffic sent to the relay. It handles the core relaying
// process itself, and can be extended using plugins to add additional
// functionality.
type Handler struct {
	config    *RelayConfig
	plugins   []Plugin
	transport *http.Transport
}

func NewHandler(config *RelayConfig, trafficPlugins []Plugin) *Handler {
	return &Handler{
		config:  config,
		plugins: trafficPlugins,
		transport: &http.Transport{
			TLSClientConfig: &tls.Config{},
			Proxy:           http.ProxyFromEnvironment,
			IdleConnTimeout: 2 * time.Second, // TODO set from configs
		},
	}
}

func (handler *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	// Drop all cookies; because the relay generally runs in a first-party
	// context, the risk of receiving cookies intended for other services is
	// high, so relaying them is a potential privacy and security risk. (In
	// cases where a particular cookie is known to be safe, it can be
	// allowlisted using the Cookies plugin.)
	originalCookieHeaders := append([]string{}, request.Header.Values("Cookie")...)
	request.Header.Del("Cookie")

	// Rewrite the request URL to point to the relay target. Plugins may change
	// these values to direct certain requests differently.
	originalURL := *request.URL
	request.URL.Scheme = handler.config.TargetScheme
	request.URL.Host = handler.config.TargetHost
	request.Host = handler.config.TargetHost

	serviced := false
	for _, trafficPlugin := range handler.plugins {
		if trafficPlugin.HandleRequest(response, request, RequestInfo{
			OriginalCookieHeaders: originalCookieHeaders,
			OriginalURL:           &originalURL,
			Serviced:              serviced,
		}) {
			serviced = true
		}
	}

	if handler.HandleRequest(response, request, serviced) {
		serviced = true
	}

	if serviced {
		logger.Printf("%s %s %s: serviced", request.Method, request.Host, request.URL)
	} else {
		logger.Printf("%s %s %s: not serviced", request.Method, request.Host, request.URL)
		http.NotFound(response, request)
	}
}

func (handler *Handler) HandleRequest(clientResponse http.ResponseWriter, clientRequest *http.Request, serviced bool) bool {
	if serviced {
		return false
	}

	if !clientRequest.URL.IsAbs() {
		http.Error(clientResponse, fmt.Sprintf("Cannot respond to relative (non-absolute) requests: %v", clientRequest.URL), 500)
		return true
	}

	handler.addRelayHeaders(clientRequest)

	if clientRequest.Header.Get("Upgrade") == "websocket" {
		return handler.handleUpgrade(clientResponse, clientRequest)
	} else {
		return handler.handleHttp(clientResponse, clientRequest)
	}
}

func (handler *Handler) addRelayHeaders(clientRequest *http.Request) {
	// Add X-Forwarded-* headers
	remoteAddrTokens := strings.Split(clientRequest.RemoteAddr, ":")
	clientRequest.Header.Add("X-Forwarded-For", remoteAddrTokens[0])
	if len(remoteAddrTokens) > 0 {
		clientRequest.Header.Add("X-Forwarded-Port", remoteAddrTokens[1])
	}
	clientRequest.Header.Add("X-Forwarded-Proto", strings.ToLower(strings.Split(clientRequest.Proto, "/")[0]))

	// Add X-Relay-Version header
	clientRequest.Header.Add(RelayVersionHeaderName, version.RelayRelease)
}

func (handler *Handler) handleHttp(clientResponse http.ResponseWriter, clientRequest *http.Request) bool {
	targetResponse, err := handler.transport.RoundTrip(clientRequest)
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

	if targetResponse.ContentLength > handler.config.MaxBodySize {
		clientResponse.WriteHeader(http.StatusServiceUnavailable)
		clientResponse.Write([]byte("Response body content-length was too large"))
	} else if targetResponse.ContentLength > 0 {
		clientResponse.WriteHeader(targetResponse.StatusCode)
		if _, err := io.CopyN(clientResponse, targetResponse.Body, targetResponse.ContentLength); err != nil {
			logger.Printf("Error relaying response body to client: %s", err)
		}
	} else if targetResponse.ContentLength < 0 {
		clientResponse.WriteHeader(targetResponse.StatusCode)
		if _, err := io.CopyN(clientResponse, targetResponse.Body, handler.config.MaxBodySize); err != nil {
			logger.Printf("Error relaying response body with unknown content-length: %s", err)
		}
	} else {
		clientResponse.WriteHeader(targetResponse.StatusCode)
	}
	return true
}

func (handler *Handler) handleUpgrade(clientResponse http.ResponseWriter, clientRequest *http.Request) bool {
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
