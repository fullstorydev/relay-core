package relay

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

type Config struct {
	Service *ServiceConfig
	Relay   *traffic.RelayConfig
}

func ReadConfig(env *commands.Environment) (*Config, error) {
	config := &Config{
		Service: NewDefaultServiceConfig(),
		Relay:   traffic.NewDefaultRelayConfig(),
	}

	if err := env.ParseRequired("RELAY_PORT", func(key string, value string) error {
		if parsedPort, err := strconv.ParseInt(value, 10, 32); err != nil {
			return err
		} else {
			config.Service.Port = int(parsedPort)
			return nil
		}
	}); err != nil {
		return nil, err
	}

	if err := env.ParseRequired("TRAFFIC_RELAY_TARGET", func(key string, value string) error {
		if targetURL, err := url.Parse(value); err != nil {
			return err
		} else if targetURL.Scheme == "" || targetURL.Host == "" {
			return fmt.Errorf("Invalid or relative target URL")
		} else {
			config.Relay.TargetScheme = targetURL.Scheme
			config.Relay.TargetHost = targetURL.Host
			return nil
		}
	}); err != nil {
		return nil, err
	}

	if err := env.ParseOptional("TRAFFIC_RELAY_MAX_BODY_SIZE", func(key string, value string) error {
		if maxBodySize, err := strconv.ParseInt(value, 10, 64); err != nil {
			return err
		} else {
			config.Relay.MaxBodySize = maxBodySize
			return nil
		}
	}); err != nil {
		return nil, err
	}

	return config, nil
}
