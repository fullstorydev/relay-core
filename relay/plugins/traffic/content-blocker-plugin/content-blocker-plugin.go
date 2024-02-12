// This plugin blocks content matching a regular expression from incoming
// request bodies and headers. See the default 'relay.yaml' for configuration
// examples.
//
// Two kinds of blocking are possible: an Exclude rule totally deletes
// content, while a Mask rule replaces it with asterisks. Although Exclude rules
// are more thorough, MASK has two benefits:
//
// - It makes it more clear that something was blocked.
// - It does not change the positions of characters, which makes it less likely
// to interfere with deserialization of the request body when complex encodings
// are used.
//
// Whether these benefits are more important than the thoroughness of using an
// Exclude rule will depend on the application.
//
// It's important to understand that this plugin does not understand the format
// of the requests it processes; it simply treats the entire request body as
// text. This makes it robust to request format changes, but it also means that
// using a regular expression that matches JSON, HTML, or CSS syntax may corrupt
// the request, so be careful.

package content_blocker_plugin

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"

	"github.com/fullstorydev/relay-core/relay/config"
	"github.com/fullstorydev/relay-core/relay/traffic"
	"github.com/fullstorydev/relay-core/relay/version"
)

var (
	Factory    contentBlockerPluginFactory
	pluginName = "block-content"
	logger     = log.New(os.Stdout, fmt.Sprintf("[traffic-%s] ", pluginName), 0)

	PluginVersionHeaderName = "X-Relay-Content-Blocker-Version"
)

type ConfigBlockRule struct {
	Exclude string
	Mask    string
}

type contentBlockerPluginFactory struct{}

func (f contentBlockerPluginFactory) Name() string {
	return pluginName
}

func (f contentBlockerPluginFactory) New(configSection *config.Section) (traffic.Plugin, error) {
	plugin := &contentBlockerPlugin{}

	addRules := func(contentKind string, rules []ConfigBlockRule) error {
		blockers := []*contentBlocker{}

		for _, rule := range rules {
			if rule.Exclude == "" && rule.Mask == "" {
				return fmt.Errorf(`Block rule must include an Exclude or Mask property`)
			}
			if rule.Exclude != "" && rule.Mask != "" {
				return fmt.Errorf(`Block rule may not include both Exclude and Mask properties`)
			}

			pattern := rule.Exclude
			mode := excludeMode
			if pattern == "" {
				pattern = rule.Mask
				mode = maskMode
			}

			if regexp, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf(`could not compile regular expression "%v": %v`, pattern, err)
			} else {
				logger.Printf("Added rule: %s %s content matching \"%s\"", mode, contentKind, regexp)
				blockers = append(blockers, &contentBlocker{
					mode:   mode,
					regexp: regexp,
				})
			}
		}

		switch contentKind {
		case "body":
			plugin.bodyBlockers = append(plugin.bodyBlockers, blockers...)
		case "header":
			plugin.headerBlockers = append(plugin.headerBlockers, blockers...)
		default:
			return fmt.Errorf(`unexpected content kind %s`, contentKind)
		}

		return nil
	}

	if err := config.ParseOptional(configSection, "body", addRules); err != nil {
		return nil, err
	}
	if err := config.ParseOptional(configSection, "header", addRules); err != nil {
		return nil, err
	}

	if err := config.ParseOptional(
		configSection,
		"TRAFFIC_EXCLUDE_BODY_CONTENT",
		func(key string, value string) error {
			return addRules("body", []ConfigBlockRule{{Exclude: value}})
		},
	); err != nil {
		return nil, err
	}
	if err := config.ParseOptional(
		configSection,
		"TRAFFIC_MASK_BODY_CONTENT",
		func(key string, value string) error {
			return addRules("body", []ConfigBlockRule{{Mask: value}})
		},
	); err != nil {
		return nil, err
	}
	if err := config.ParseOptional(
		configSection,
		"TRAFFIC_EXCLUDE_HEADER_CONTENT",
		func(key string, value string) error {
			return addRules("header", []ConfigBlockRule{{Exclude: value}})
		},
	); err != nil {
		return nil, err
	}
	if err := config.ParseOptional(
		configSection,
		"TRAFFIC_MASK_HEADER_CONTENT",
		func(key string, value string) error {
			return addRules("header", []ConfigBlockRule{{Mask: value}})
		},
	); err != nil {
		return nil, err
	}

	if len(plugin.bodyBlockers) == 0 && len(plugin.headerBlockers) == 0 {
		return nil, nil
	}

	return plugin, nil
}

type contentBlockerPlugin struct {
	bodyBlockers   []*contentBlocker
	headerBlockers []*contentBlocker
}

func (plug contentBlockerPlugin) Name() string {
	return pluginName
}

func (plug contentBlockerPlugin) HandleRequest(
	response http.ResponseWriter,
	request *http.Request,
	info traffic.RequestInfo,
) bool {
	if info.Serviced {
		return false
	}

	if serviced := plug.blockHeaderContent(response, request); serviced {
		return true
	}
	if serviced := plug.blockBodyContent(response, request); serviced {
		return true
	}

	// Tag the request with a header for debugging purposes.
	request.Header.Add(PluginVersionHeaderName, version.RelayRelease)

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

	processedBody, err := io.ReadAll(request.Body)
	if err != nil {
		http.Error(response, fmt.Sprintf("Error reading request body: %s", err), 500)
		request.Body = http.NoBody
		return true
	}

	for _, blocker := range plug.bodyBlockers {
		processedBody = blocker.Block(processedBody)
	}

	// If the length of the body has changed, we should update the
	// Content-Length header too.
	contentLength := int64(len(processedBody))
	if contentLength != request.ContentLength {
		request.ContentLength = contentLength
		request.Header.Set("Content-Length", strconv.FormatInt(contentLength, 10))
	}

	request.Body = io.NopCloser(bytes.NewBuffer(processedBody))
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

var maskSymbol = []byte("*")

// contentBlocker applies a content blocking transformation (either exclude or
// mask) to content that matches a regular expression.
type contentBlocker struct {
	mode   contentBlockerMode
	regexp *regexp.Regexp
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
		panic(fmt.Errorf("invalid content blocking mode: %v", b.mode))
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
