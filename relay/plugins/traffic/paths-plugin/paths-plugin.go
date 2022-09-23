// This plugin watches incoming traffic and optionally rewrites request URL
// paths. The most common use is to remove or rewrite a path prefix.

package paths_plugin

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/fullstorydev/relay-core/relay/config"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

var (
	Factory    pathsPluginFactory
	pluginName = "paths"
	logger     = log.New(os.Stdout, fmt.Sprintf("[traffic-%s] ", pluginName), 0)
)

type ConfigRouteRule struct {
	Path       string
	TargetPath string `yaml:"target-path"`
	TargetUrl  string `yaml:"target-url"`
}

type pathsPluginFactory struct{}

func (f pathsPluginFactory) Name() string {
	return pluginName
}

func (f pathsPluginFactory) New(configSection *config.Section) (traffic.Plugin, error) {
	plugin := &pathsPlugin{}

	addRules := func(_ string, rules []ConfigRouteRule) error {
		for _, rule := range rules {
			if rule.TargetPath == "" && rule.TargetUrl == "" {
				return fmt.Errorf(`Route for path "%v" has no target`, rule.Path)
			}
			if rule.TargetPath != "" && rule.TargetUrl != "" {
				return fmt.Errorf(`Route for path "%v" has multiple targets`, rule.Path)
			}

			replacement := rule.TargetPath
			target := pathTarget
			if replacement == "" {
				replacement = rule.TargetUrl
				target = urlTarget
			}

			if match, err := regexp.Compile(rule.Path); err != nil {
				return fmt.Errorf(`Could not compile path regular expression "%v": %v`, rule.Path, err)
			} else {
				logger.Printf(`Added rule: route "%s" to %s "%s"`, match, target, replacement)
				plugin.rules = append(plugin.rules, &pathRule{
					match:       match,
					replacement: replacement,
					target:      target,
				})
			}
		}

		return nil
	}

	if err := config.ParseOptional(configSection, "routes", addRules); err != nil {
		return nil, err
	}
	if err := f.addTrafficPathsRule(configSection, addRules); err != nil {
		return nil, err
	}
	if err := f.addSpecialsRule(configSection, addRules); err != nil {
		return nil, err
	}

	if len(plugin.rules) == 0 {
		return nil, nil
	}

	return plugin, nil
}

// addTrafficPathsRule reads a path rule defined using the TRAFFIC_PATHS_MATCH
// and TRAFFIC_PATHS_REPLACEMENT options, if one exists. These rules match
// against the path portion of a URL and, if a match is found, replace the URL's
// path. (The other portions of the URL remain the same.)
func (f pathsPluginFactory) addTrafficPathsRule(
	configSection *config.Section,
	addRules func(_ string, rules []ConfigRouteRule) error,
) error {
	// Read the replacement value, which is just a literal string. If it's not
	// present, the rule wouldn't do anything, regardless of whether
	// TRAFFIC_PATHS_MATCH is present, so we just return.
	replacement, err := config.LookupOptional[string](configSection, "TRAFFIC_PATHS_REPLACEMENT")
	if err != nil {
		return err
	}
	if replacement == nil {
		return nil
	}

	// Read the match value, which is a Go regular expression. Since we know a
	// replacement value was specified, we treat the match value as required;
	// one doesn't make sense without the other.
	return config.ParseRequired(
		configSection,
		"TRAFFIC_PATHS_MATCH",
		func(key string, match string) error {
			return addRules(key, []ConfigRouteRule{{
				Path:       match,
				TargetPath: *replacement,
			}})
		},
	)
}

// addSpecialsRule reads path rules defined using the TRAFFIC_RELAY_SPECIALS
// option, if one exists. These rules match against the path portion of a URL
// and, if a match is found, replace the entire URL. (Query params are left
// untouched.)
func (f pathsPluginFactory) addSpecialsRule(
	configSection *config.Section,
	addRules func(_ string, rules []ConfigRouteRule) error,
) error {
	return config.ParseOptional(
		configSection,
		"TRAFFIC_RELAY_SPECIALS",
		func(key string, value string) error {
			specialsTokens := strings.Split(value, " ")
			if len(specialsTokens)%2 != 0 {
				return fmt.Errorf("Last key has no value")
			}

			var rules []ConfigRouteRule
			for i := 0; i < len(specialsTokens); i += 2 {
				rules = append(rules, ConfigRouteRule{
					Path:      specialsTokens[i],
					TargetUrl: specialsTokens[i+1],
				})
			}

			return addRules(key, rules)
		},
	)
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
