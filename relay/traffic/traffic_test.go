package traffic_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/test-interceptor-plugin"
	"github.com/fullstorydev/relay-core/relay/test"
	"github.com/fullstorydev/relay-core/relay/traffic"
	"github.com/fullstorydev/relay-core/relay/version"
	"golang.org/x/net/websocket"
)

func TestBasicRelay(t *testing.T) {
	test.WithCatcherAndRelay(t, nil, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		catcherBody := getBody(catcherService.HttpUrl(), t)
		if catcherBody == nil {
			return
		}

		relayBody := getBody(relayService.HttpUrl(), t)
		if relayBody == nil {
			return
		}

		if bytes.Equal(catcherBody, relayBody) == false {
			t.Errorf("Bodies don't match: \"%v\" \"%v\"", catcherBody, relayBody)
			return
		}
	})
}

func TestRelayedCookies(t *testing.T) {
	testCases := []struct {
		desc                  string
		env                   map[string]string
		originalCookieHeaders []string
		expectedCookieHeaders []string
	}{
		{
			desc:                  "No cookies are relayed by default",
			env:                   map[string]string{},
			originalCookieHeaders: []string{"SPECIAL_ID=298zf09hf012fh2; token=u32t4o3tb3gg43", "_gat=1"},
			expectedCookieHeaders: nil,
		},
		{
			desc: "Multiple Cookie headers are merged",
			env: map[string]string{
				"TRAFFIC_RELAY_COOKIES": "SPECIAL_ID token _gat",
			},
			originalCookieHeaders: []string{"SPECIAL_ID=298zf09hf012fh2; token=u32t4o3tb3gg43", "_gat=1"},
			expectedCookieHeaders: []string{"SPECIAL_ID=298zf09hf012fh2; token=u32t4o3tb3gg43; _gat=1"},
		},
		{
			desc: "Only allowlisted cookies are relayed",
			env: map[string]string{
				"TRAFFIC_RELAY_COOKIES": "SPECIAL_ID foo _gat",
			},
			originalCookieHeaders: []string{"SPECIAL_ID=298zf09hf012fh2; token=u32t4o3tb3gg43; foo=bar", "_gat=1; bar=foo"},
			expectedCookieHeaders: []string{"SPECIAL_ID=298zf09hf012fh2; foo=bar; _gat=1"},
		},
		{
			desc: "A Cookie header is dropped entirely when no cookies match",
			env: map[string]string{
				"TRAFFIC_RELAY_COOKIES": "bar",
			},
			originalCookieHeaders: []string{"SPECIAL_ID=298zf09hf012fh2; token=u32t4o3tb3gg43; foo=bar", "_gat=1; bar=foo"},
			expectedCookieHeaders: []string{"bar=foo"},
		},
	}

	for _, testCase := range testCases {
		test.WithCatcherAndRelay(t, testCase.env, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
			request, err := http.NewRequest("GET", relayService.HttpUrl(), nil)
			if err != nil {
				t.Errorf("Test '%v': Error creating request: %v", testCase.desc, err)
				return
			}

			for _, cookieHeaderValue := range testCase.originalCookieHeaders {
				request.Header.Add("Cookie", cookieHeaderValue)
			}

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Errorf("Test '%v': Error GETing: %v", testCase.desc, err)
				return
			}
			defer response.Body.Close()

			if response.StatusCode != 200 {
				t.Errorf("Test '%v': Expected 200 response: %v", testCase.desc, response)
				return
			}

			lastRequest, err := catcherService.LastRequest()
			if err != nil {
				t.Errorf("Test '%v': Error reading last request from catcher: %v", testCase.desc, err)
				return
			}

			actualCookieHeaders := lastRequest.Header["Cookie"]
			if !reflect.DeepEqual(testCase.expectedCookieHeaders, actualCookieHeaders) {
				t.Errorf(
					"Test '%v': Expected Cookie header values '%v' but got '%v'",
					testCase.desc,
					testCase.expectedCookieHeaders,
					actualCookieHeaders,
				)
			}
		})
	}
}

func TestRelayedHeaders(t *testing.T) {
	testCases := []struct {
		desc            string
		env             map[string]string
		originalHeaders map[string]string
		expectedHeaders map[string]string
	}{
		{
			desc: "Headers are relayed by default",
			env:  map[string]string{},
			originalHeaders: map[string]string{
				"Accept-Encoding": "deflate, gzip;q=1.0, *;q=0.5",
				"Downlink":        "100",
				"Origin":          "https://test.com",
				"Viewport-Width":  "100",
			},
			expectedHeaders: map[string]string{
				"Accept-Encoding": "deflate, gzip;q=1.0, *;q=0.5",
				"Downlink":        "100",
				"Origin":          "https://test.com",
				"Viewport-Width":  "100",
			},
		},
		{
			desc: "Overriding the Origin header works",
			env: map[string]string{
				"TRAFFIC_RELAY_ORIGIN_OVERRIDE": "example.com",
			},
			originalHeaders: map[string]string{
				"Accept-Encoding": "deflate, gzip;q=1.0, *;q=0.5",
				"Downlink":        "100",
				"Origin":          "https://test.com",
				"Viewport-Width":  "100",
			},
			expectedHeaders: map[string]string{
				"Accept-Encoding": "deflate, gzip;q=1.0, *;q=0.5",
				"Downlink":        "100",
				"Origin":          "http://example.com",
				"Viewport-Width":  "100",
			},
		},
	}

	for _, testCase := range testCases {
		var lastClientIP, lastClientPort string

		plugins := []traffic.PluginFactory{
			test_interceptor_plugin.NewFactoryWithListener(func(request *http.Request) {
				// Capture the actual IP and port used in the request.
				addrComponents := strings.Split(request.RemoteAddr, ":")
				lastClientIP = addrComponents[0]
				lastClientPort = addrComponents[1]
			}),
		}

		test.WithCatcherAndRelay(t, testCase.env, plugins, func(catcherService *catcher.Service, relayService *relay.Service) {
			request, err := http.NewRequest("GET", relayService.HttpUrl(), nil)
			if err != nil {
				t.Errorf("Test '%v': Error creating request: %v", testCase.desc, err)
				return
			}

			for headerName, headerValue := range testCase.originalHeaders {
				request.Header.Add(headerName, headerValue)
			}

			response, err := http.DefaultClient.Do(request)
			if err != nil {
				t.Errorf("Test '%v': Error GETing: %v", testCase.desc, err)
				return
			}
			defer response.Body.Close()

			if response.StatusCode != 200 {
				t.Errorf("Test '%v': Expected 200 response: %v", testCase.desc, response)
				return
			}

			lastRequest, err := catcherService.LastRequest()
			if err != nil {
				t.Errorf("Test '%v': Error reading last request from catcher: %v", testCase.desc, err)
				return
			}

			// Check for the built-in headers that the relay always generates.
			testCase.expectedHeaders["X-Forwarded-Proto"] = "http"
			testCase.expectedHeaders[traffic.RelayVersionHeaderName] = version.RelayRelease
			testCase.expectedHeaders["X-Forwarded-For"] = lastClientIP
			testCase.expectedHeaders["X-Forwarded-Port"] = lastClientPort

			for headerName, expectedHeaderValue := range testCase.expectedHeaders {
				expectedHeaderValues := []string{expectedHeaderValue}
				actualHeaderValues := lastRequest.Header[headerName]
				if !reflect.DeepEqual(expectedHeaderValues, actualHeaderValues) {
					t.Errorf(
						"Test '%v': Expected '%v' header values '%v' but got '%v'",
						testCase.desc,
						headerName,
						expectedHeaderValues,
						actualHeaderValues,
					)
				}
			}
		})
	}
}

func TestMaxBodySize(t *testing.T) {
	env := map[string]string{
		"TRAFFIC_RELAY_MAX_BODY_SIZE": fmt.Sprintf("%v", 5),
	}

	test.WithCatcherAndRelay(t, env, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		response, err := http.Get(relayService.HttpUrl())
		if err != nil {
			t.Errorf("Error GETing: %v", err)
			return
		}
		defer response.Body.Close()
		if response.StatusCode != 503 {
			t.Errorf("Expected 503 response for surpassing max body size: %v", response)
			return
		}
	})
}

func TestRelayNotFound(t *testing.T) {
	test.WithCatcherAndRelay(t, nil, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		faviconURL := fmt.Sprintf("%v/favicon.ico", relayService.HttpUrl())
		response, err := http.Get(faviconURL)
		if err != nil {
			t.Errorf("Error GETing: %v", err)
			return
		}
		if response.StatusCode != 404 {
			t.Errorf("Should have received 404: %v", response)
			return
		}
	})
}

func TestWebSocketEcho(t *testing.T) {
	test.WithCatcherAndRelay(t, nil, nil, func(catcherService *catcher.Service, relayService *relay.Service) {
		echoURL := fmt.Sprintf("%v/echo", relayService.WsUrl())
		ws, err := websocket.Dial(echoURL, "", relayService.HttpUrl())
		if err != nil {
			t.Errorf("Error dialing websocket: %v", err)
			return
		}
		err = testEcho(ws, "Come in, good buddy")
		if err != nil {
			t.Errorf("Error in echo: %v", err)
			return
		}
		err = testEcho(ws, "10-4, Rocket")
		if err != nil {
			t.Errorf("Error in second echo: %v", err)
			return
		}
	})
}

func testEcho(conn *websocket.Conn, message string) error {
	_, err := conn.Write([]byte(message))
	if err != nil {
		return err
	}
	var response = make([]byte, len(message)+10)
	n, err := conn.Read(response)
	if err != nil {
		return err
	}
	if strings.Compare(message, string(response[:n])) != 0 {
		return errors.New(fmt.Sprintf("Unexpected echo response: %v", string(response[:n])))
	}
	return nil
}

func getBody(url string, t *testing.T) []byte {
	response, err := http.Get(url)
	if err != nil {
		t.Errorf("Error GETing: %v", err)
		return nil
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Errorf("Non-200 GET: %v", response)
		return nil
	}
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Errorf("Error GETing body: %v", err)
		return nil
	}
	return body
}
