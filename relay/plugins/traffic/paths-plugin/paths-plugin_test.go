package paths_plugin_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/paths-plugin"
	"github.com/fullstorydev/relay-core/relay/test"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

func TestPathRewriting(t *testing.T) {
	testCases := []pathsPluginTestCase{
		{
			desc: "Basic path remapping works",
			env: map[string]string{
				"TRAFFIC_PATHS_MATCH":       `^/foo/`,
				"TRAFFIC_PATHS_REPLACEMENT": `/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/foo/bar/baz`,
			expectedUrl: `${TARGET_HTTP_URL}/xyz/bar/baz`,
		},
		{
			desc: "Paths that do not match are not changed",
			env: map[string]string{
				"TRAFFIC_PATHS_MATCH":       `^/foo/`,
				"TRAFFIC_PATHS_REPLACEMENT": `/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/abc/bar/baz`,
			expectedUrl: `${TARGET_HTTP_URL}/abc/bar/baz`,
		},
		{
			desc: "Capture groups can be used",
			env: map[string]string{
				"TRAFFIC_PATHS_MATCH":       `^/([^/]*)/foo/([^/]*)/bar/`,
				"TRAFFIC_PATHS_REPLACEMENT": `/$1/xyz/$2/abc/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/apple/foo/banana/bar/carrot`,
			expectedUrl: `${TARGET_HTTP_URL}/apple/xyz/banana/abc/carrot`,
		},
		{
			desc: "Query params are preserved",
			env: map[string]string{
				"TRAFFIC_PATHS_MATCH":       `^/foo/`,
				"TRAFFIC_PATHS_REPLACEMENT": `/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/foo/bar/baz?x=y&abc=123`,
			expectedUrl: `${TARGET_HTTP_URL}/xyz/bar/baz?x=y&abc=123`,
		},
		{
			desc: "Matching and replacement only affect the path",
			env: map[string]string{
				"TRAFFIC_PATHS_MATCH":       `foo`,
				"TRAFFIC_PATHS_REPLACEMENT": `xyz`,
			},
			originalUrl: `${RELAY_HTTP_URL}/foo/bar/baz?x=y&foo=123`,
			expectedUrl: `${TARGET_HTTP_URL}/xyz/bar/baz?x=y&foo=123`,
		},
	}

	for _, testCase := range testCases {
		runPathsPluginTest(t, testCase)
	}
}

func TestSpecialPaths(t *testing.T) {
	testCases := []pathsPluginTestCase{
		{
			desc: "Basic path remapping works",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/foo/ ${ALT_TARGET_HTTP_URL}/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/foo/bar/baz`,
			expectedUrl: `${ALT_TARGET_HTTP_URL}/xyz/bar/baz`,
		},
		{
			desc: "Paths that do not match are not changed",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/foo/ ${ALT_TARGET_HTTP_URL}/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/abc/bar/baz`,
			expectedUrl: `${TARGET_HTTP_URL}/abc/bar/baz`,
		},
		{
			desc: "Capture groups can be used",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/([^/]*)/foo/([^/]*)/bar/ ${ALT_TARGET_HTTP_URL}/$1/xyz/$2/abc/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/apple/foo/banana/bar/carrot`,
			expectedUrl: `${ALT_TARGET_HTTP_URL}/apple/xyz/banana/abc/carrot`,
		},
		{
			desc: "Query params are preserved",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/foo/ ${ALT_TARGET_HTTP_URL}/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/foo/bar/baz?x=y&abc=123`,
			expectedUrl: `${ALT_TARGET_HTTP_URL}/xyz/bar/baz?x=y&abc=123`,
		},
		{
			desc: "Matching and replacement only affect the path",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/foo/ ${ALT_TARGET_HTTP_URL}/xyz/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/foo/bar/baz?x=y&foo=123`,
			expectedUrl: `${ALT_TARGET_HTTP_URL}/xyz/bar/baz?x=y&foo=123`,
		},
		{
			desc: "Multiple rules can be used at once (part 1)",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/apple/ ${ALT_TARGET_HTTP_URL}/xyz/ ^/banana/ ${ALT_TARGET_HTTP_URL}/abc/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/apple/foo/bar`,
			expectedUrl: `${ALT_TARGET_HTTP_URL}/xyz/foo/bar`,
		},
		{
			desc: "Multiple rules can be used at once (part 2)",
			env: map[string]string{
				"TRAFFIC_RELAY_SPECIALS": `^/apple/ ${ALT_TARGET_HTTP_URL}/xyz/ ^/banana/ ${ALT_TARGET_HTTP_URL}/abc/`,
			},
			originalUrl: `${RELAY_HTTP_URL}/banana/foo/bar`,
			expectedUrl: `${ALT_TARGET_HTTP_URL}/abc/foo/bar`,
		},
	}

	for _, testCase := range testCases {
		runPathsPluginTest(t, testCase)
	}
}

type pathsPluginTestCase struct {
	desc        string
	env         map[string]string
	originalUrl string
	expectedUrl string
}

func runPathsPluginTest(t *testing.T, testCase pathsPluginTestCase) {
	plugins := []traffic.PluginFactory{
		paths_plugin.Factory,
	}

	// Start our own instance of the catcher service. This allows us to provide
	// an alternative target for test cases to redirect to.
	altCatcherService := catcher.NewService()
	if err := altCatcherService.Start("localhost", 0); err != nil {
		t.Errorf("Error starting alternative catcher: %v", err)
		return
	}
	defer altCatcherService.Close()

	// Substitute ALT_TARGET_HTTP_URL into the environment variables so it can
	// be used in TRAFFIC_RELAY_SPECIALS.
	env := map[string]string{}
	for envVar, value := range testCase.env {
		env[envVar] = strings.Replace(value, "${ALT_TARGET_HTTP_URL}", altCatcherService.HttpUrl(), -1)
	}

	test.WithCatcherAndRelay(t, env, plugins, func(catcherService *catcher.Service, relayService *relay.Service) {
		// Substitute RELAY_HTTP_URL and TARGET_HTTP_URL into the URLs. We
		// unfortunately can't combine this with the environment variable
		// substitution because we only have access to these values once the
		// relay has started up, but at that point the plugins have already been
		// configured and changes to environment variable values would have no
		// effect.
		varReplacer := strings.NewReplacer(
			"${ALT_TARGET_HTTP_URL}", altCatcherService.HttpUrl(),
			"${RELAY_HTTP_URL}", relayService.HttpUrl(),
			"${TARGET_HTTP_URL}", catcherService.HttpUrl(),
		)
		originalUrl := varReplacer.Replace(testCase.originalUrl)
		expectedUrl := varReplacer.Replace(testCase.expectedUrl)

		response, err := http.Get(originalUrl)
		if err != nil {
			t.Errorf("Error GETing: %v", err)
			return
		}
		defer response.Body.Close()

		if response.StatusCode != 200 {
			t.Errorf("Test '%v': Expected 200 response: %v", testCase.desc, response)
			return
		}

		lastRequest, err := catcherService.LastRequest()
		if err != nil {
			lastRequest, err = altCatcherService.LastRequest()
		}
		if err != nil {
			t.Errorf("Error reading last request from catcher: %v", err)
			return
		}

		// The value of lastRequest.URL is relative; convert it to an absolute URL.
		actualUrl := *lastRequest.URL
		actualUrl.Scheme = "http"
		actualUrl.Host = lastRequest.Host

		if actualUrl.String() != expectedUrl {
			t.Errorf(
				"Test '%v': Expected URL '%v' to become '%v' but got: %v",
				testCase.desc,
				originalUrl,
				expectedUrl,
				actualUrl.String(),
			)
		}
	})
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
