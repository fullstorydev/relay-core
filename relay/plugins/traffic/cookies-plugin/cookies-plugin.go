// This plugin provides the capability to allowlist cookies on incoming
// requests. By default, all cookies are blocked. This is because in the context
// of the relay, cookies are quite high-risk; it usually runs in a first-party
// context, so the risk of receiving cookies that were intended for another
// service is substantial.

package cookies_plugin

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fullstorydev/relay-core/relay/config"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    cookiesPluginFactory
	pluginName = "cookies"
	logger     = log.New(os.Stdout, fmt.Sprintf("[traffic-%s] ", pluginName), 0)
)

type cookiesPluginFactory struct{}

func (f cookiesPluginFactory) Name() string {
	return pluginName
}

func (f cookiesPluginFactory) New(configSection *config.Section) (traffic.Plugin, error) {
	plugin := &cookiesPlugin{
		allowlist: map[string]bool{},
	}

	if err := config.ParseOptional(
		configSection,
		"allowlist",
		func(key string, allowlist []string) error {
			for _, cookieName := range allowlist {
				logger.Printf(`Added rule: allowlist cookie "%s"`, cookieName)
				plugin.allowlist[cookieName] = true
			}

			return nil
		},
	); err != nil {
		return nil, err
	}

	if err := config.ParseOptional(
		configSection,
		"TRAFFIC_RELAY_COOKIES",
		func(key string, allowlist string) error {
			for _, cookieName := range strings.Split(allowlist, " ") {
				logger.Printf(`Added rule: allowlist cookie "%s"`, cookieName)
				plugin.allowlist[cookieName] = true
			}

			return nil
		},
	); err != nil {
		return nil, err
	}

	if len(plugin.allowlist) == 0 {
		return nil, nil
	}

	return plugin, nil
}

type cookiesPlugin struct {
	allowlist map[string]bool // The name of cookies that should be relayed.
}

func (plug cookiesPlugin) Name() string {
	return pluginName
}

func (plug cookiesPlugin) HandleRequest(
	response http.ResponseWriter,
	request *http.Request,
	info traffic.RequestInfo,
) bool {
	if info.Serviced {
		return false
	}

	// Restore the original Cookie header so that we can parse it using the
	// methods on http.Request.
	for _, headerValue := range info.OriginalCookieHeaders {
		request.Header.Add("Cookie", headerValue)
	}

	// Parse the Cookie header and filter out cookies which aren't present in
	// the allowlist.
	var cookies []string
	for _, cookie := range request.Cookies() {
		if !plug.allowlist[cookie.Name] {
			continue
		}
		cookies = append(cookies, cookie.String())
	}

	// Reserialize the Cookie header.
	request.Header.Set("Cookie", strings.Join(cookies, "; "))

	return false
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
