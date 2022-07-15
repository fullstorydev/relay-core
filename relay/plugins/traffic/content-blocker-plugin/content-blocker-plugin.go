// The Content-Blocker plugin blocks content matching a regular expression from
// incoming request bodies and headers. It's controlled by four environment
// variables:
//
// - TRAFFIC_EXCLUDE_BODY_CONTENT: If set, a regular expression that will be
// evaluated against request bodies. Matching content will be completely deleted
// from the request.
// - TRAFFIC_MASK_BODY_CONTENT: If set, a regular expression that will be
// evaluated against request bodies. Matching content will be replaced with
// asterisks.
// - TRAFFIC_EXCLUDE_HEADER_CONTENT: Like TRAFFIC_EXCLUDE_BODY_CONTENT, but
// applies to header values.
// - TRAFFIC_MASK_HEADER_CONTENT: Like TRAFFIC_MASK_BODY_CONTENT, but applies to
// header values.
//
// Although EXCLUDE is more thorough, MASK has two benefits:
//
// - It makes it more clear that something was blocked.
// - It does not change the positions of characters, which makes it less likely
// to interfere with deserialization of the request body when complex encodings
// are used.
//
// Whether these benefits are more important than the thoroughness of EXCLUDE
// will depend on the application.
//
// It's important to understand that this plugin does not understand the format
// of the requests it processes; it simply treats the entire request body as
// text. This makes it robust to request format changes, but it also means that
// using a regular expression that matches JSON, HTML, or CSS syntax may corrupt
// the request, so be careful.
//
// Because this plugin transforms requests, it must run before the relay plugin.
package content_blocker_plugin

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/fullstorydev/relay-core/relay/plugins/traffic"
)

var (
	Factory    contentBlockerPluginFactory
	logger     = log.New(os.Stdout, "[traffic-content-blocker] ", 0)
	pluginName = "Content-Blocker"

	excludeBodyContentVar   = "TRAFFIC_EXCLUDE_BODY_CONTENT"   // A go regexp string or empty
	maskBodyContentVar      = "TRAFFIC_MASK_BODY_CONTENT"      // A go regexp string or empty
	excludeHeaderContentVar = "TRAFFIC_EXCLUDE_HEADER_CONTENT" // A go regexp string or empty
	maskHeaderContentVar    = "TRAFFIC_MASK_HEADER_CONTENT"    // A go regexp string or empty
)

type contentBlockerPluginFactory struct{}

func (f contentBlockerPluginFactory) Name() string {
	return pluginName
}

func (f contentBlockerPluginFactory) ConfigVars() map[string]bool {
	return map[string]bool{
		excludeBodyContentVar:   false,
		maskBodyContentVar:      false,
		excludeHeaderContentVar: false,
		maskHeaderContentVar:    false,
	}
}

func (f contentBlockerPluginFactory) New(env map[string]string) (traffic.Plugin, error) {
	bodyBlockers, err := newContentBlockerList(
		env,
		"body",
		excludeBodyContentVar,
		maskBodyContentVar,
	)
	if err != nil {
		return nil, err
	}

	headerBlockers, err := newContentBlockerList(
		env,
		"header",
		excludeHeaderContentVar,
		maskHeaderContentVar,
	)
	if err != nil {
		return nil, err
	}

	if len(bodyBlockers) == 0 && len(headerBlockers) == 0 {
		return nil, nil
	}

	return &contentBlockerPlugin{
		bodyBlockers:   bodyBlockers,
		headerBlockers: headerBlockers,
	}, nil
}

type contentBlockerPlugin struct {
	bodyBlockers   []*contentBlocker
	headerBlockers []*contentBlocker
}

func (plug contentBlockerPlugin) Name() string {
	return pluginName
}

func (plug contentBlockerPlugin) HandleRequest(response http.ResponseWriter, request *http.Request, serviced bool) bool {
	if serviced := plug.blockHeaderContent(response, request); serviced {
		return true
	}
	if serviced := plug.blockBodyContent(response, request); serviced {
		return true
	}
	return false
}

func (plug contentBlockerPlugin) blockHeaderContent(response http.ResponseWriter, request *http.Request) bool {
	if len(plug.headerBlockers) == 0 {
		return false
	}

	for _, headerValues := range request.Header {
		for i, headerValue := range headerValues {
			processedValue := []byte(headerValue)
			for _, blocker := range plug.headerBlockers {
				processedValue = blocker.Block(processedValue)
			}
			headerValues[i] = string(processedValue)
		}
	}

	return false
}

func (plug contentBlockerPlugin) blockBodyContent(response http.ResponseWriter, request *http.Request) bool {
	if len(plug.bodyBlockers) == 0 {
		return false
	}

	// Block all websocket connections if we're blocking body content.
	// TODO(sethf): This is necessary because the current plugin architecture
	// doesn't give us a hook that would allow us to handle websocket messages,
	// so we wouldn't be able to actually do any blocking. The safest thing to
	// do for now is to fail closed. In the short term, this won't do any harm,
	// because we don't actually need to support websockets, but if that changes
	// we'll need to revisit this.
	if len(plug.bodyBlockers) > 0 && request.Header.Get("Upgrade") == "websocket" {
		logger.Println("Rejecting websocket connection (content blocking is not supported with websockets):", request.URL)
		http.Error(response, fmt.Sprintf("Blocking unsupported websocket connection: %v", request.URL), 500)
		return true
	}

	if request.Body == nil || request.Body == http.NoBody {
		return false
	}

	processedBody, err := ioutil.ReadAll(request.Body)
	if err != nil {
		http.Error(response, fmt.Sprintf("Error reading request body: %s", err), 500)
		request.Body = http.NoBody
		return true
	}
	initialLength := len(processedBody)

	for _, blocker := range plug.bodyBlockers {
		processedBody = blocker.Block(processedBody)
	}

	// If the length of the body has changed, we should update the
	// Content-Length header too.
	finalLength := len(processedBody)
	if finalLength != initialLength {
		contentLength := int64(finalLength)
		request.ContentLength = contentLength
		request.Header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}

	request.Body = ioutil.NopCloser(bytes.NewBuffer(processedBody))
	return false
}

type contentBlockerMode int64

const (
	maskMode contentBlockerMode = iota
	excludeMode
)

func (mode contentBlockerMode) String() string {
	switch mode {
	case maskMode:
		return "mask"
	case excludeMode:
		return "exclude"
	default:
		return "(unknown mode)"
	}
}

// newContentBlockerList is a helper that creates a group of related
// contentBlocker instances as a single operation.
func newContentBlockerList(
	env map[string]string,
	listName string,
	excludeVar string,
	maskVar string,
) ([]*contentBlocker, error) {
	var blockers []*contentBlocker

	if blocker, err := newContentBlocker(env, excludeVar, excludeMode); err != nil {
		return nil, err
	} else if blocker != nil {
		blockers = append(blockers, blocker)
	}

	if blocker, err := newContentBlocker(env, maskVar, maskMode); err != nil {
		return nil, err
	} else if blocker != nil {
		blockers = append(blockers, blocker)
	}

	for _, blocker := range blockers {
		logger.Printf("Content-blocker plugin will %s %s content matching \"%s\"", blocker.mode, listName, blocker.regexp)
	}

	return blockers, nil
}

var maskSymbol = []byte("*")

// contentBlocker applies a content blocking transformation (either exclude or
// mask) to content that matches a regular expression.
type contentBlocker struct {
	mode   contentBlockerMode
	regexp *regexp.Regexp
}

func newContentBlocker(
	env map[string]string,
	varName string,
	mode contentBlockerMode,
) (*contentBlocker, error) {
	pattern := env[varName]
	if pattern == "" {
		return nil, nil
	}

	regexp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("Could not compile %v regular expression: %v", varName, err)
	}

	return &contentBlocker{
		mode:   mode,
		regexp: regexp,
	}, nil
}

func (b *contentBlocker) Block(content []byte) []byte {
	switch b.mode {
	case maskMode:
		return b.regexp.ReplaceAllFunc(content, func(matched []byte) []byte {
			return bytes.Repeat(maskSymbol, len(matched))
		})
	case excludeMode:
		return b.regexp.ReplaceAllLiteral(content, []byte{})
	default:
		panic(fmt.Errorf("Invalid content blocking mode: %v", b.mode))
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
