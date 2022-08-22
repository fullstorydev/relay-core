package commands

import (
	"bufio"
	"errors"
	"log"
	"os"
	"strings"
)

var logger = log.New(os.Stdout, "[relay] ", 0)

type EnvVar struct {
	EnvKey     string
	Required   bool
	DefaultVal string
}

type Environment map[string]string

// EnvironmentProvider is an interface used to retrieve a string-based,
// key-value set of configuration options.
type EnvironmentProvider interface {
	// Read attempts to read the values of the provided variables from the
	// underlying source. If a value for a variable is found, it's written into
	// the provided Environment.
	Read(vars []EnvVar, env Environment)
}

// GetEnvironmentOrPrintUsage reads the requested variables from an environment
// provider. If a required variable is not found, usage information is printed
// to stdout and an error is returned; otherwise, a populated Environment
// containing the values of the requested variables is returned.
func GetEnvironmentOrPrintUsage(
	provider EnvironmentProvider,
	vars []EnvVar,
) (Environment, error) {
	env := Environment{}
	setupDefaultValues(vars, env)
	provider.Read(vars, env)

	if err := checkEnvironment(vars, env); err != nil {
		printEnvUsage(vars, env)
		return nil, err
	}

	return env, nil
}

func setupDefaultValues(vars []EnvVar, env Environment) {
	for _, variable := range vars {
		if variable.DefaultVal == "" {
			continue
		}
		env[variable.EnvKey] = variable.DefaultVal
	}
}

func checkEnvironment(vars []EnvVar, env Environment) error {
	for _, variable := range vars {
		if variable.Required == false {
			continue
		}

		envVal, found := env[variable.EnvKey]
		if found == false || len(envVal) == 0 {
			return errors.New("Required environment variable is missing: " + variable.EnvKey)
		}
	}

	return nil
}

func printEnvUsage(vars []EnvVar, env Environment) {
	logger.Println("Required configuration variables via .env or environment:")
	for _, variable := range vars {
		if variable.Required == false {
			continue
		}
		envVal, found := env[variable.EnvKey]
		if found == false || len(envVal) == 0 {
			logger.Println("\t" + variable.EnvKey + ": missing")
		} else {
			logger.Println("\t" + variable.EnvKey)
		}
	}
	logger.Println("")

	logger.Println("Optional environment variables:")
	for _, variable := range vars {
		if variable.Required {
			continue
		}
		envVal, found := env[variable.EnvKey]
		if found == false || len(envVal) == 0 {
			logger.Println("\t" + variable.EnvKey + ": missing")
		} else {
			logger.Println("\t" + variable.EnvKey)
		}
	}
}

// DefaultEnvironmentProvider tries to read environment variables from various
// external sources. In order of precedence:
//  * OS environment variables.
//  * Values from any .env file that may exist.
type DefaultEnvironmentProvider struct {
}

func NewDefaultEnvironmentProvider() EnvironmentProvider {
	return &DefaultEnvironmentProvider{}
}

func (provider *DefaultEnvironmentProvider) Read(vars []EnvVar, env Environment) {
	provider.readDotEnv(vars, env)
	provider.readEnvironment(vars, env)
}

// readDotEnv reads environment variables from a .env file, if one is present,
// and adds their values to env. Returns an error if reading from .env fails.
func (provider *DefaultEnvironmentProvider) readDotEnv(vars []EnvVar, env Environment) error {
	dotEnvVals, err := provider.parseDotEnv(".env")
	if err != nil {
		return err
	}
	for key, value := range dotEnvVals {
		env[key] = value
	}
	return nil
}

func (provider *DefaultEnvironmentProvider) readEnvironment(vars []EnvVar, env Environment) {
	for _, variable := range vars {
		envVal, found := os.LookupEnv(variable.EnvKey)
		if found && len(envVal) > 0 {
			env[variable.EnvKey] = envVal
		}
	}
}

func (provider *DefaultEnvironmentProvider) parseDotEnv(filePath string) (Environment, error) {
	results := Environment{}

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
	env Environment
}

func NewTestEnvironmentProvider(env Environment) EnvironmentProvider {
	return &TestEnvironmentProvider{
		env: env,
	}
}

func (provider *TestEnvironmentProvider) Read(vars []EnvVar, env Environment) {
	for _, variable := range vars {
		envVal, found := provider.env[variable.EnvKey]
		if found {
			env[variable.EnvKey] = envVal
		}
	}
}
