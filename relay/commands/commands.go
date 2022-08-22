package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

var logger = log.New(os.Stdout, "[relay] ", 0)

// Environment exposes a set of convenience methods for reading configuration
// values from an EnvironmentProvider.
type Environment struct {
	provider EnvironmentProvider
}

func NewEnvironment(provider EnvironmentProvider) *Environment {
	return &Environment{
		provider: provider,
	}
}

// Get returns the value associated with the provided key, if present, and the
// empty string otherwise.
func (env *Environment) Get(key string) string {
	val, _ := env.provider.Lookup(key)
	return val
}

// LookupOptional returns the value associated with the provided key, if
// present, and a boolean indicating whether the key was found.
func (env *Environment) LookupOptional(key string) (string, bool) {
	return env.provider.Lookup(key)
}

// LookupOptional returns the value associated with the provided key, if
// present. If the key is not present, an error is returned.
func (env *Environment) LookupRequired(key string) (string, error) {
	val, ok := env.provider.Lookup(key)
	if ok {
		return val, nil
	}
	return "", fmt.Errorf("Missing required configuration variable: %v", key)
}

// ParseOptional invokes a callback with the value of the provided key, if it's
// present, and propagates any error the callback returns. If the key is not
// found, the callback is not invoked and no error is reported.
func (env *Environment) ParseOptional(
	key string,
	action func(key string, value string) error,
) error {
	value, ok := env.LookupOptional(key)
	if !ok {
		return nil
	}

	return action(key, value)
}

// ParseOptional invokes a callback with the value of the provided key, if it's
// present, and propagates any error the callback returns. If the key is not
// found, an error is reported.
func (env *Environment) ParseRequired(
	key string,
	action func(key string, value string) error,
) error {
	value, err := env.LookupRequired(key)
	if err != nil {
		return err
	}

	if err := action(key, value); err != nil {
		return fmt.Errorf(`Error parsing configuration variable %s ("%s"): %v`, key, value, err)
	}

	return nil
}

// EnvironmentProvider is an interface used to retrieve a string-based,
// key-value set of configuration options.
type EnvironmentProvider interface {
	// Lookup returns the value associated with the provided key, if present,
	// and a boolean indicating whether the key was found.
	Lookup(key string) (string, bool)
}

// DefaultEnvironmentProvider tries to read environment variables from various
// external sources. In order of precedence:
//  * OS environment variables.
//  * Values from any .env file that may exist.
type DefaultEnvironmentProvider struct {
	dotEnv map[string]string
}

func NewDefaultEnvironmentProvider() (EnvironmentProvider, error) {
	dotEnv, err := parseDotEnv(".env")
	if err != nil {
		return nil, err
	}

	return &DefaultEnvironmentProvider{
		dotEnv: dotEnv,
	}, nil
}

func (provider *DefaultEnvironmentProvider) Lookup(key string) (string, bool) {
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

func parseDotEnv(filePath string) (map[string]string, error) {
	results := map[string]string{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
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

	return results, nil
}

// TestEnvironmentProvider reads environment variables from a hard-coded list.
type TestEnvironmentProvider struct {
	env map[string]string
}

func NewTestEnvironmentProvider(env map[string]string) EnvironmentProvider {
	return &TestEnvironmentProvider{
		env: env,
	}
}

func (provider *TestEnvironmentProvider) Lookup(key string) (string, bool) {
	envVal, ok := provider.env[key]
	if ok && len(envVal) > 0 {
		return envVal, true
	}

	return "", false
}
