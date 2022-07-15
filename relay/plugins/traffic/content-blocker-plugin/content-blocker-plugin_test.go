package content_blocker_plugin_test

import (
	"bytes"
	"net/http"
	"strconv"
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/content-blocker-plugin"
	"github.com/fullstorydev/relay-core/relay/test"
)

func TestContentBlockerBlocksContent(t *testing.T) {
	testCases := []contentBlockerTestCase{
		{
			desc: "Body content can be excluded",
			env: commands.Environment{
				"TRAFFIC_EXCLUDE_BODY_CONTENT": `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
			},
			originalBody: `{ "content": "Excluded IP address = 215.1.0.335." }`,
			expectedBody: `{ "content": "Excluded IP address = ." }`,
		},
		{
			desc: "Body content can be masked",
			env: commands.Environment{
				"TRAFFIC_MASK_BODY_CONTENT": `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
			},
			originalBody: `{ "content": "Excluded IP address = 215.1.0.335." }`,
			expectedBody: `{ "content": "Excluded IP address = ***********." }`,
		},
		{
			desc: "Header content can be excluded",
			env: commands.Environment{
				"TRAFFIC_EXCLUDE_HEADER_CONTENT": `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
			},
			originalHeaders: map[string]string{
				"X-Forwarded-For": "foo.com,192.168.0.1,bar.com",
			},
			expectedHeaders: map[string]string{
				"X-Forwarded-For": "foo.com,,bar.com",
			},
		},
		{
			desc: "Header content can be masked",
			env: commands.Environment{
				"TRAFFIC_MASK_HEADER_CONTENT": `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
			},
			originalHeaders: map[string]string{
				"X-Forwarded-For": "foo.com,192.168.0.1,bar.com",
			},
			expectedHeaders: map[string]string{
				"X-Forwarded-For": "foo.com,***********,bar.com",
			},
		},
		{
			desc: "Header values are blocked but header names are not",
			env: commands.Environment{
				"TRAFFIC_EXCLUDE_HEADER_CONTENT": `(?i)BAR`,
				"TRAFFIC_MASK_HEADER_CONTENT":    `(?i)FOO`,
			},
			originalHeaders: map[string]string{
				"X-Barrier":  "foo bar baz",
				"X-Football": "foo bar baz",
			},
			expectedHeaders: map[string]string{
				"X-Barrier":  "***  baz",
				"X-Football": "***  baz",
			},
		},
		{
			desc: "Exclusion takes priority over masking",
			env: commands.Environment{
				"TRAFFIC_EXCLUDE_BODY_CONTENT": `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
				"TRAFFIC_MASK_BODY_CONTENT":    `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
			},
			originalBody: `{ "content": "Excluded IP address = 215.1.0.335." }`,
			expectedBody: `{ "content": "Excluded IP address = ." }`,
		},
		{
			desc: "Complex configurations are supported",
			env: commands.Environment{
				"TRAFFIC_EXCLUDE_BODY_CONTENT":   `(?i)EXCLUDED`,
				"TRAFFIC_MASK_BODY_CONTENT":      `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
				"TRAFFIC_EXCLUDE_HEADER_CONTENT": `(?i)DELETED`,
				"TRAFFIC_MASK_HEADER_CONTENT":    `(foo|bar)`,
			},
			originalBody: `{ "content": "Excluded, deleted foo bar IP address = 215.1.0.335." }`,
			expectedBody: `{ "content": ", deleted foo bar IP address = ***********." }`,
			originalHeaders: map[string]string{
				"X-Forwarded-For":  "192.168.0.1",
				"X-Headerfoobar":   "bar foo baz bar baz foobar",
				"X-Special-Header": "Some EXCLUDED, DELETED content",
			},
			expectedHeaders: map[string]string{
				"X-Forwarded-For":  "192.168.0.1",
				"X-Headerfoobar":   "*** *** baz *** baz ******",
				"X-Special-Header": "Some EXCLUDED,  content",
			},
		},
	}

	for _, testCase := range testCases {
		runContentBlockerTest(t, testCase)
	}
}

func TestContentBlockerBlocksWebsockets(t *testing.T) {
	env := commands.Environment{
		"TRAFFIC_MASK_BODY_CONTENT": `[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+`,
	}
	plugins := map[string]bool{
		content_blocker_plugin.Factory.Name(): true,
	}

	test.WithCatcherAndRelay(t, env, plugins, func(catcherService *catcher.Service, relayService *relay.Service) {
		request, err := http.NewRequest(
			"POST",
			relayService.HttpUrl(),
			bytes.NewBufferString(`{ "content": "192.168.0.1" }`),
		)
		if err != nil {
			t.Errorf("Error creating request: %v", err)
			return
		}

		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Upgrade", "websocket")

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Errorf("Error POSTing: %v", err)
			return
		}
		defer response.Body.Close()

		// This plugin doesn't support websockets, so we should fail closed and
		// the attempt to establish a websocket connection should fail.
		if response.StatusCode != 500 {
			t.Errorf("Expected 500 response: %v", response)
			return
		}
	})
}

type contentBlockerTestCase struct {
	desc            string
	env             commands.Environment
	originalBody    string
	expectedBody    string
	originalHeaders map[string]string
	expectedHeaders map[string]string
}

func runContentBlockerTest(t *testing.T, testCase contentBlockerTestCase) {
	plugins := map[string]bool{
		content_blocker_plugin.Factory.Name(): true,
	}

	originalHeaders := testCase.originalHeaders
	if originalHeaders == nil {
		originalHeaders = make(map[string]string)
	}

	expectedHeaders := testCase.expectedHeaders
	if expectedHeaders == nil {
		expectedHeaders = make(map[string]string)
	}

	test.WithCatcherAndRelay(t, testCase.env, plugins, func(catcherService *catcher.Service, relayService *relay.Service) {
		request, err := http.NewRequest(
			"POST",
			relayService.HttpUrl(),
			bytes.NewBufferString(testCase.originalBody),
		)
		if err != nil {
			t.Errorf("Test '%v': Error creating request: %v", testCase.desc, err)
			return
		}

		request.Header.Set("Content-Type", "application/json")
		for header, headerValue := range originalHeaders {
			request.Header.Set(header, headerValue)
		}

		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Errorf("Test '%v': Error POSTing: %v", testCase.desc, err)
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

		for expectedHeader, expectedHeaderValue := range expectedHeaders {
			actualHeaderValue := lastRequest.Header.Get(expectedHeader)
			if expectedHeaderValue != actualHeaderValue {
				t.Errorf(
					"Test '%v': Expected header '%v' with value '%v' but got: %v",
					testCase.desc,
					expectedHeader,
					expectedHeaderValue,
					actualHeaderValue,
				)
			}
		}

		lastRequestBody, err := catcherService.LastRequestBody()
		if err != nil {
			t.Errorf("Test '%v': Error reading last request body from catcher: %v", testCase.desc, err)
			return
		}

		lastRequestBodyStr := string(lastRequestBody)
		if testCase.expectedBody != lastRequestBodyStr {
			t.Errorf(
				"Test '%v': Expected body '%v' but got: %v",
				testCase.desc,
				testCase.expectedBody,
				lastRequestBodyStr,
			)
		}

		contentLength, err := strconv.Atoi(lastRequest.Header.Get("Content-Length"))
		if err != nil {
			t.Errorf("Test '%v': Error parsing Content-Length: %v", testCase.desc, err)
			return
		}

		if contentLength != len(lastRequestBody) {
			t.Errorf(
				"Test '%v': Content-Length is %v but actual body length is %v",
				testCase.desc,
				contentLength,
				len(lastRequestBody),
			)
		}
	})
}
