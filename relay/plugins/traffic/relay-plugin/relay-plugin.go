package relay_plugin

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

	"github.com/fullstorydev/relay-core/relay/plugins/traffic"
)

const DefaultMaxBodySize int64 = 1024 * 2048 // 2MB

const RelayVersionHeaderName = "X-Relay-Version"
const RelayVersion = "v0.1.3" // TODO set this from tags automatically during git commit

var (
	Factory    relayPluginFactory
	logger     = log.New(os.Stdout, "[traffic-relay] ", 0)
	pluginName = "Relay"

	hasPort                       = regexp.MustCompile(`:\d+$`)
	trafficRelayTargetVar         = "TRAFFIC_RELAY_TARGET"
	trafficRelayCookiesVar        = "TRAFFIC_RELAY_COOKIES"
	trafficRelayMaxBodySizeVar    = "TRAFFIC_RELAY_MAX_BODY_SIZE"
	trafficRelayOriginOverrideVar = "TRAFFIC_RELAY_ORIGIN_OVERRIDE"
	trafficRelaySpecials          = "TRAFFIC_RELAY_SPECIALS"
)

type relayPluginFactory struct{}

func (f relayPluginFactory) Name() string {
	return pluginName
}

func (f relayPluginFactory) ConfigVars() map[string]bool {
	return map[string]bool{
		trafficRelayTargetVar:         true,
		trafficRelaySpecials:          false,
		trafficRelayCookiesVar:        false,
		trafficRelayMaxBodySizeVar:    false,
		trafficRelayOriginOverrideVar: false,
	}
}

func (f relayPluginFactory) New(env map[string]string) (traffic.Plugin, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
		Proxy:           http.ProxyFromEnvironment,
		IdleConnTimeout: 2 * time.Second, // TODO set from configs
	}
	plugin := &relayPlugin{
		transport:      transport,
		targetScheme:   "",
		targetHost:     "",
		specials:       []special{},
		originOverride: "",
		relayedCookies: map[string]bool{},
		maxBodySize:    DefaultMaxBodySize,
	}

	targetVar := env[trafficRelayTargetVar]
	targetURL, err := url.Parse(targetVar)
	if err != nil {
		return nil, fmt.Errorf("Could not parse %v environment variable URL: %v", trafficRelayTargetVar, targetVar)
	}
	plugin.targetScheme = targetURL.Scheme
	plugin.targetHost = targetURL.Host

	specialsVar := env[trafficRelaySpecials]
	if len(specialsVar) > 0 {
		specialsTokens := strings.Split(specialsVar, " ")
		if len(specialsTokens)%2 != 0 {
			return nil, fmt.Errorf("Could not parse %v environment variable: %v", trafficRelaySpecials, specialsTokens)
		}
		for i := 0; i < len(specialsTokens); i += 2 {
			matchVar := specialsTokens[i]
			replacementString := specialsTokens[i+1]
			matchRE, err := regexp.Compile(matchVar)
			if err != nil {
				return nil, fmt.Errorf("Could not compile regular expression \"%v\" in %v: %v", matchVar, trafficRelaySpecials, err)
			}
			special := special{
				matchRE,
				replacementString,
			}
			plugin.specials = append(plugin.specials, special)
			logger.Printf("Relaying special expression \"%v\" to \"%v\"", special.match, special.replacement)
		}
	}

	originOverrideVar := env[trafficRelayOriginOverrideVar]
	if len(originOverrideVar) > 0 {
		plugin.originOverride = originOverrideVar
	}

	cookiesVar := env[trafficRelayCookiesVar]
	if len(cookiesVar) > 0 {
		for _, cookieName := range strings.Split(cookiesVar, " ") { // Should we support spaces?
			plugin.relayedCookies[cookieName] = true
		}
	}

	maxBodySizeVar := env[trafficRelayMaxBodySizeVar]
	if len(maxBodySizeVar) > 0 {
		maxBodySize, err := strconv.ParseInt(maxBodySizeVar, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Error parsing TRAFFIC_RELAY_MAX_BODY_SIZE environment variable: %v", err)
		}
		plugin.maxBodySize = maxBodySize
	}

	return plugin, nil
}

type special struct {
	match       *regexp.Regexp
	replacement string
}

type relayPlugin struct {
	transport      *http.Transport
	targetScheme   string          // http|https
	targetHost     string          // e.g. 192.168.0.1:1234
	specials       []special       // path patterns that are mapped to fully qualified URLs
	originOverride string          // default to passing Origin header as-is, but set to override
	relayedCookies map[string]bool // the name of cookies that should be relayed
	maxBodySize    int64           // maximum length in bytes of relayed bodies
}

// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Forwarded
type forwarded struct {
	byHeader    string // The interface where the request came in to the proxy server.
	forHeader   string // The client that initiated the request and subsequent proxies in a chain of proxies.
	hostHeader  string // The Host request header field as received by the proxy.
	protoHeader string // Indicates which protocol was used to make the request (typically "http" or "https").
}

func (plug relayPlugin) Name() string {
	return pluginName
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

func (plug *relayPlugin) prepRelayRequest(clientRequest *http.Request) {
	clientRequest.URL.Scheme = plug.targetScheme
	clientRequest.URL.Host = plug.targetHost
	clientRequest.Host = plug.targetHost

	if len(plug.specials) > 0 {
		for _, special := range plug.specials {
			if special.match.Match([]byte(clientRequest.URL.Path)) == false {
				continue
			}
			urlVal := special.match.ReplaceAllString(clientRequest.URL.Path, special.replacement)
			newURL, err := url.Parse(urlVal)
			if err != nil {
				logger.Printf("Failed to create URL for special %v: %v", special.match, err)
			} else {
				clientRequest.URL.Scheme = newURL.Scheme
				clientRequest.URL.Host = newURL.Host
				clientRequest.Host = newURL.Host
				clientRequest.URL.Path = newURL.Path
			}
		}
	} else if len(plug.originOverride) > 0 {
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

	// Add X-Forwarded-* headers
	remoteAddrTokens := strings.Split(clientRequest.RemoteAddr, ":")
	clientRequest.Header.Add("X-Forwarded-For", remoteAddrTokens[0])
	if len(remoteAddrTokens) > 0 {
		clientRequest.Header.Add("X-Forwarded-Port", remoteAddrTokens[1])
	}
	clientRequest.Header.Add("X-Forwarded-Proto", strings.ToLower(strings.Split(clientRequest.Proto, "/")[0]))

	// Add X-Relay-Version header
	clientRequest.Header.Add(RelayVersionHeaderName, RelayVersion)
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
