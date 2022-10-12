package paths_plugin

/*
	The Paths plugin watches incoming traffic and optionally rewrites request URL paths.
	The most common use is to remove or rewrite a path prefix.
*/

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    pathsPluginFactory
	logger     = log.New(os.Stdout, "[traffic-paths] ", 0)
	pluginName = "Paths"
)

type pathsPluginFactory struct{}

func (f pathsPluginFactory) Name() string {
	return pluginName
}

func (f pathsPluginFactory) New(env *commands.Environment) (traffic.Plugin, error) {
	plugin := &pathsPlugin{}

	if rule, err := f.readTrafficPathsRule(env); err != nil {
		return nil, err
	} else if rule != nil {
		plugin.rules = append(plugin.rules, rule)
	}

	if rules, err := f.readSpecialsRule(env); err != nil {
		return nil, err
	} else {
		plugin.rules = append(plugin.rules, rules...)
	}

	if len(plugin.rules) == 0 {
		return nil, nil
	}

	for _, rule := range plugin.rules {
		logger.Printf(`Paths plugin will replace all "%s" with "%s"`, rule.match, rule.replacement)
	}

	return plugin, nil
}

// readTrafficPathsRule reads a path rule defined using the TRAFFIC_PATHS_MATCH
// and TRAFFIC_PATHS_REPLACEMENT environment variables, if one exists. These
// rules match against the path portion of a URL and, if a match is found,
// replace the URL's path. (The other portions of the URL remain the same.)
// Returns nil if no meaningful rule is defined.
func (f pathsPluginFactory) readTrafficPathsRule(env *commands.Environment) (*pathRule, error) {
	rule := &pathRule{
		target: pathTarget,
	}

	// Read the replacement value, which is just a literal string. If it's not
	// present, the rule wouldn't do anything, regardless of whether
	// TRAFFIC_PATHS_MATCH is present, so we just return.
	if replacement, ok := env.LookupOptional("TRAFFIC_PATHS_REPLACEMENT"); !ok {
		return nil, nil
	} else {
		rule.replacement = replacement
	}

	// Read the match value, which is a Go regular expression. Since we know a
	// replacement value was specified, we treat the match value as required;
	// one doesn't make sense without the other.
	if err := env.ParseRequired("TRAFFIC_PATHS_MATCH", func(key string, value string) error {
		if match, err := regexp.Compile(value); err != nil {
			return fmt.Errorf("Could not compile regular expression: %v", err)
		} else {
			rule.match = match
			return nil
		}
	}); err != nil {
		return nil, err
	}

	return rule, nil
}

// readSpecialsRule reads path rules defined using the TRAFFIC_RELAY_SPECIALS
// environment variable, if one exists. These rules match against the path
// portion of a URL and, if a match is found, replace the entire URL. (Query
// params are left untouched.) Returns an empty list if no such rules are
// defined.
func (f pathsPluginFactory) readSpecialsRule(env *commands.Environment) ([]*pathRule, error) {
	var rules []*pathRule

	if err := env.ParseOptional("TRAFFIC_RELAY_SPECIALS", func(key string, value string) error {
		specialsTokens := strings.Split(value, " ")
		if len(specialsTokens)%2 != 0 {
			return fmt.Errorf("Last key has no value")
		}

		for i := 0; i < len(specialsTokens); i += 2 {
			matchVar := specialsTokens[i]
			replacement := specialsTokens[i+1]

			match, err := regexp.Compile(matchVar)
			if err != nil {
				return fmt.Errorf("Could not compile regular expression \"%v\": %v", matchVar, err)
			}

			rules = append(rules, &pathRule{
				match:       match,
				replacement: replacement,
				target:      urlTarget,
			})
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return rules, nil
}

type pathsPlugin struct {
	rules []*pathRule
}

type pathRule struct {
	match       *regexp.Regexp
	replacement string
	target      pathRuleTarget
}

type pathRuleTarget int64

const (
	pathTarget pathRuleTarget = iota
	urlTarget
)

func (target pathRuleTarget) String() string {
	switch target {
	case pathTarget:
		return "path"
	case urlTarget:
		return "URL"
	default:
		return "(unknown target)"
	}
}

func (plug pathsPlugin) Name() string {
	return pluginName
}

func (plug pathsPlugin) HandleRequest(
	response http.ResponseWriter,
	request *http.Request,
	info traffic.RequestInfo,
) bool {
	if info.Serviced {
		return false
	}

	for _, rule := range plug.rules {
		switch rule.target {
		case pathTarget:
			// If there's a match, replace the requested URL's path.
			request.URL.Path = rule.match.ReplaceAllString(request.URL.Path, rule.replacement)

		case urlTarget:
			// If the rule matches the requested URL's path...
			if rule.match.Match([]byte(request.URL.Path)) == false {
				break
			}

			// ...then replace the *entire URL, except for query params*. The
			// path is provided as an input to ReplaceAllString() so that the
			// replacement can reference capture groups from the path.
			urlVal := rule.match.ReplaceAllString(request.URL.Path, rule.replacement)
			newURL, err := url.Parse(urlVal)
			if err != nil {
				logger.Printf("Failed to create URL for path rule %v: %v", rule.match, err)
			} else {
				request.URL.Scheme = newURL.Scheme
				request.URL.Host = newURL.Host
				request.Host = newURL.Host
				request.URL.Path = newURL.Path
			}
		}
	}

	return false
}

/*
Copyright 2020 FullStory, Inc.

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
