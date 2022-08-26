package headers_plugin_test

import (
	"net/http"
	"reflect"
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/headers-plugin"
	"github.com/fullstorydev/relay-core/relay/test"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

func TestHeadersPlugin(t *testing.T) {
	testCases := []struct {
		desc            string
		env             map[string]string
		originalHeaders map[string]string
		expectedHeaders map[string]string
	}{
		{
			desc: "The Origin header is relayed unchanged by default",
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

	plugins := []traffic.PluginFactory{
		headers_plugin.Factory,
	}

	for _, testCase := range testCases {
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

/*
Copyright 2022 FullStory, Inc.

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
