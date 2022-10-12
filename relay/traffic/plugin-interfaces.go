package traffic

import (
	"net/http"
	"net/url"

	"github.com/fullstorydev/relay-core/relay/commands"
)

// PluginFactory is the interface that the relay uses to create plugin
// instances.
type PluginFactory interface {
	// Name returns a human readable name for this plugin, like "Logging" or
	// "Attack detector".
	Name() string

	// New configures and returns an instance of this plugin, or an error if
	// configuration failed. Configuration options are read from the given
	// environment.
	//
	// Factories may return nil if the plugin should be inactive given the
	// provided configuration.
	New(env *commands.Environment) (Plugin, error)
}

// Plugin is the interface exposed by plugin instances.
type Plugin interface {
	// Name returns a human readable name for this plugin, like "Logging" or
	// "Attack detector". This should match the value returned by the
	// corresponding PluginFactory#Name().
	Name() string

	// HandleRequest is invoked to allow a plugin to handle an incoming traffic
	// HTTP request.
	//
	// Plugins may ignore an incoming request, alter it in some way, or service
	// the request and return a response to the client.
	//
	// HandleRequest should return true if a response has been sent to the
	// client.
	HandleRequest(
		response http.ResponseWriter,
		request *http.Request,
		requestInfo RequestInfo,
	) bool
}

// RequestInfo provides additional information about incoming requests.
type RequestInfo struct {
	// The original cookie headers included in the client request. For security
	// and privacy reasons, these are automatically removed from the client
	// request before plugins get an opportunity to handle it.
	OriginalCookieHeaders []string

	// The original URL requested by the client, before any redirection by the
	// relay.
	OriginalURL *url.URL

	// If true, a response has already been sent to the client.
	Serviced bool
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
