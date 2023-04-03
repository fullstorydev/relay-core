package environment

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	logger = log.New(os.Stdout, "[relay] ", 0)

	// Matches "${FOO}", "${FOO:BAR}", "$(FOO)", or "$(FOO:BAR)".
	varSubstitutionRegexp = regexp.MustCompile(`(\\*)((\$\{([^:}]*)(:([^}]*))?})|(\$\(([^:)]*)(:([^)]*))?\)))`)

	// Regular expressions matching YAML primitive values, taken from the YAML
	// spec: https://yaml.org/spec/1.2.2/#103-core-schema
	nullValueRegexp  = regexp.MustCompile(`^(null|Null|NULL|~|)$`)
	boolValueRegexp  = regexp.MustCompile(`^(true|True|TRUE|false|False|FALSE)$`)
	intValueRegexp   = regexp.MustCompile(`^([-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+)$`)
	floatValueRegexp = regexp.MustCompile(`^([-+]?(\.[0-9]+|[0-9]+(\.[0-9]*)?)([eE][-+]?[0-9]+)?|[-+]?(\.inf|\.Inf|\.INF)|\.nan|\.NaN|\.NAN)$`)
)

// Map exposes a set of convenience methods for reading configuration
// values from an EnvironmentProvider.
type Map struct {
	provider Provider
}

func NewMap(provider Provider) *Map {
	return &Map{
		provider: provider,
	}
}

// Get returns the value associated with the provided key, if present, and the
// empty string otherwise.
func (env *Map) Get(key string) string {
	val, _ := env.provider.Lookup(key)
	return val
}

// LookupOptional returns the value associated with the provided key, if
// present, and a boolean indicating whether the key was found.
func (env *Map) LookupOptional(key string) (string, bool) {
	return env.provider.Lookup(key)
}

// LookupRequired returns the value associated with the provided key, if
// present. If the key is not present, an error is returned.
func (env *Map) LookupRequired(key string) (string, error) {
	val, ok := env.provider.Lookup(key)
	if ok {
		return val, nil
	}
	return "", fmt.Errorf("Missing required configuration variable: %v", key)
}

// SubstituteVars substitutes variables from the environment into the provided
// YAML string. "${FOO}" is replaced with the value of the variable "FOO".
// "${FOO:BAR}" is also replaced by the value of "FOO", but if "FOO" is not
// present, the value "BAR" is used instead. If "$(FOO)" or "$(FOO:BAR")" is
// used, no escaping is performed before substitution.
func (env *Map) SubstituteVarsIntoYaml(input string) string {
	return varSubstitutionRegexp.ReplaceAllStringFunc(input, func(expression string) string {
		submatches := varSubstitutionRegexp.FindStringSubmatch(expression)

		var envVar string
		var defaultValue string
		var escapeValue func(input string) string

		// Count the number of backslashes that appear before this substitution
		// expression. If the number is odd, this expression was escaped; all we
		// need to do is drop the backslash that was "consumed" by escaping this
		// expression and return everything else just as it appeared in the
		// source test.
		var backslashes = submatches[1]
		if len(backslashes)%2 == 1 {
			return submatches[0][1:]
		}

		// There were an even number of backslashes; this substitution
		// expression wasn't escaped. We'll perform the substitution and just
		// pass through the backslashes as-is. The code below will call this
		// function to actually do the substitution once we determine what the
		// value to be substituted is and which algorithm to use to escape that
		// value.
		substituteValue := func(value string) string {
			return backslashes + escapeValue(value)
		}

		if submatches[3] != "" {
			// We've got ${VAR} or ${VAR:DEFAULT}.
			envVar = submatches[4]
			defaultValue = submatches[6]

			// For this kind of substitution, we attempt to autodetect the
			// appropriate YAML type for the value and transform it. Note that
			// the result is always a "primitive" YAML value, and never an
			// arbitrary hunk of YAML syntax.
			escapeValue = func(value string) string {
				// Leave values that are some kind of non-string YAML primitive
				// unchanged.
				if nullValueRegexp.MatchString(value) ||
					boolValueRegexp.MatchString(value) ||
					intValueRegexp.MatchString(value) ||
					floatValueRegexp.MatchString(value) {
					return value
				}

				// Default to treating this value as a string. We pass it
				// through yaml.Marshal() to ensure that it's correctly escaped.
				if yamlBytes, err := yaml.Marshal(&value); err == nil {
					// yaml.Marshal() will insert a newline after the literal
					// value it generates, so we need to remove it.
					return strings.TrimRight(string(yamlBytes), "\n")
				}

				// The input is invalid; just return the empty string.
				logger.Printf(`Invalid value for environment variable '%v': %v`, envVar, value)
				return ""
			}
		} else {
			// We've got $(VAR) or $(VAR:DEFAULT).
			envVar = submatches[8]
			defaultValue = submatches[10]

			// For this kind of substitution, we just substitute in the value
			// directly, without any transformations. This is usually not
			// desirable because it requires you to format your environment
			// variables as a valid YAML value and deal with quotes and escape
			// sequences, but it gives the user full control over the
			// substitution process.
			escapeValue = func(value string) string {
				return value
			}
		}

		// As a special case, if the variable name is the empty string, just
		// return the default value. The default value may also be empty, in
		// which case the whole thing evaluates to the empty string.
		if envVar == "" {
			return substituteValue(defaultValue)
		}

		// Substitute in the value of the variable from the environment. If the
		// variable is not found, the default value is used if available.
		// Otherwise, the result is the empty string.
		if value, ok := env.LookupOptional(envVar); !ok {
			return substituteValue(defaultValue)
		} else {
			return substituteValue(value)
		}
	})
}

// Provider is an interface used to retrieve a string-based,
// key-value set of configuration options.
type Provider interface {
	// Lookup returns the value associated with the provided key, if present,
	// and a boolean indicating whether the key was found.
	Lookup(key string) (string, bool)
}

// DefaultProvider tries to read environment variables from various
// external sources. In order of precedence:
//   - OS environment variables.
//   - Values from any .env file that may exist.
type DefaultProvider struct {
	dotEnv map[string]string
}

func NewDefaultProvider() Provider {
	return &DefaultProvider{
		dotEnv: parseDotEnv(".env"),
	}
}

func (provider *DefaultProvider) Lookup(key string) (string, bool) {
	envVal, ok := os.LookupEnv(key)
	if ok && len(envVal) > 0 {
		return envVal, true
	}

	envVal, ok = provider.dotEnv[envVal]
	if ok && len(envVal) > 0 {
		return envVal, true
	}

	return "", false
}

func parseDotEnv(filePath string) map[string]string {
	results := map[string]string{}

	file, err := os.Open(filePath)
	if err != nil {
		// It's OK for .env to not exist.
		return results
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " 	")
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		separatorIndex := strings.Index(line, "=")
		if separatorIndex == -1 || separatorIndex == len(line)-1 {
			logger.Println("Invalid dotenv line:", line)
			continue
		}
		key := strings.Trim(line[0:separatorIndex], " 	")
		value := strings.Trim(line[separatorIndex+1:], " 	")
		if strings.HasPrefix(value, "\"") {
			value = value[1:]
		}
		if strings.HasSuffix(value, "\"") {
			value = value[0 : len(value)-1]
		}
		results[key] = value
	}

	return results
}
