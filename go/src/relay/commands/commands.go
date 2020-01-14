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

/*
SetupEnvironment tries to set environment variables using .env then current env variables.
It returns an error if one or more expected variables are not present from at least on of those sources.
*/
func SetupEnvironment(vars []EnvVar, dirs []string) error {
	setupDotEnv(vars)
	setupDefaultValues(vars)
	return CheckEnvironment(vars, dirs)
}

func CheckEnvironment(vars []EnvVar, dirs []string) error {
	for _, variable := range vars {
		if variable.Required == false {
			continue
		}
		envVal, found := os.LookupEnv(variable.EnvKey)
		if found == false || len(envVal) == 0 {
			return errors.New("Required environment variable is missing: " + variable.EnvKey)
		}
	}

	for _, envKey := range dirs {
		if checkDir(os.Getenv(envKey)) != nil {
			logger.Println("Could not read " + envKey + " directory: " + os.Getenv(envKey))
			return errors.New("Invalid directory: " + os.Getenv(envKey))
		}
	}

	return nil
}

func PrintEnvUsage(vars []EnvVar) {
	logger.Println("Required configuration variables via .env or environment:")
	for _, variable := range vars {
		if variable.Required == false {
			continue
		}
		envVal, found := os.LookupEnv(variable.EnvKey)
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
		envVal, found := os.LookupEnv(variable.EnvKey)
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

// Set .env values to process env values if process env key doesn't already exist
func setupDotEnv(vars []EnvVar) error {
	dotEnvVals, err := parseDotEnv(".env")
	if err != nil {
		return err
	}
	for key, value := range dotEnvVals {
		_, found := os.LookupEnv(key)
		if found {
			continue
		}
		os.Setenv(key, value)
	}
	return nil
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

func setupDefaultValues(vars []EnvVar) {
	for _, variable := range vars {
		if variable.DefaultVal == "" {
			continue
		}
		envVal, found := os.LookupEnv(variable.EnvKey)
		if found && len(envVal) != 0 {
			continue
		}
		os.Setenv(variable.EnvKey, variable.DefaultVal)
	}
}
