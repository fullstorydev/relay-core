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
	IsDir      bool
}

type Environment map[string]string

// GetEnvironment tries to read environment variables from various sources. In
// order of precedence:
//  * OS environment variables.
//  * Values from any .env file that may exist.
//  * Default values specified by the caller.
// It returns an error if one or more expected variables are not present in any
// of the sources.
func GetEnvironment(vars []EnvVar) (Environment, error) {
	env := map[string]string{}
	setupDefaultValues(vars, env)
	readDotEnv(vars, env)
	readEnvironment(vars, env)
	return env, checkEnvironment(vars, env)
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

		if variable.IsDir && checkDir(envVal) != nil {
			logger.Println("Could not read " + variable.EnvKey + " directory: " + envVal)
			return errors.New("Invalid directory: " + envVal)
		}
	}

	return nil
}

func PrintEnvUsage(vars []EnvVar, env Environment) {
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

func checkDir(dirPath string) error {
	pathInfo, err := os.Stat(dirPath)
	if err != nil {
		return err
	}
	if pathInfo.IsDir() == false {
		return errors.New("Not a directory")
	}
	return nil
}

// readDotEnv reads environment variables from a .env file, if one is present,
// and adds their values to env. Returns an error if reading from .env fails.
func readDotEnv(vars []EnvVar, env Environment) error {
	dotEnvVals, err := parseDotEnv(".env")
	if err != nil {
		return err
	}
	for key, value := range dotEnvVals {
		env[key] = value
	}
	return nil
}

func readEnvironment(vars []EnvVar, env Environment) {
	for _, variable := range vars {
		envVal, found := os.LookupEnv(variable.EnvKey)
		if found && len(envVal) > 0 {
			env[variable.EnvKey] = envVal
		}
	}
}

func parseDotEnv(filePath string) (Environment, error) {
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

func setupDefaultValues(vars []EnvVar, env Environment) {
	for _, variable := range vars {
		if variable.DefaultVal == "" {
			continue
		}
		env[variable.EnvKey] = variable.DefaultVal
	}
}
