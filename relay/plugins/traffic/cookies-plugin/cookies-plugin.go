package cookies_plugin

// The Cookies plugin provides the capability to allowlist cookies on incoming
// requests. By default, all cookies are blocked. This is because in the context
// of the relay, cookies are quite high-risk; it usually runs in a first-party
// context, so the risk of receiving cookies that were intended for another
// service is substantial.

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    cookiesPluginFactory
	logger     = log.New(os.Stdout, "[traffic-cookies] ", 0)
	pluginName = "Cookies"
)

type cookiesPluginFactory struct{}

func (f cookiesPluginFactory) Name() string {
	return pluginName
}

func (f cookiesPluginFactory) New(env *commands.Environment) (traffic.Plugin, error) {
	plugin := &cookiesPlugin{
		allowlist: map[string]bool{},
	}

	if cookiesVal, ok := env.LookupOptional("TRAFFIC_RELAY_COOKIES"); ok {
		for _, cookieName := range strings.Split(cookiesVal, " ") {
			logger.Printf(`Cookies plugin will allowlist cookie "%s"`, cookieName)
			plugin.allowlist[cookieName] = true
		}
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
