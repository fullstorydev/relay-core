package relay

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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
		} else {
			config.Relay.TargetScheme = targetURL.Scheme
			config.Relay.TargetHost = targetURL.Host
			return nil
		}
	}); err != nil {
		return nil, err
	}

	if cookiesVar, ok := env.LookupOptional("TRAFFIC_RELAY_COOKIES"); ok {
		for _, cookieName := range strings.Split(cookiesVar, " ") { // Should we support spaces?
			config.Relay.RelayedCookies[cookieName] = true
		}
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

	if originOverrideVar, ok := env.LookupOptional("TRAFFIC_RELAY_ORIGIN_OVERRIDE"); ok {
		config.Relay.OriginOverride = originOverrideVar
	}

	if err := env.ParseOptional("TRAFFIC_RELAY_SPECIALS", func(key string, value string) error {
		specialsTokens := strings.Split(value, " ")
		if len(specialsTokens)%2 != 0 {
			return fmt.Errorf("Last key has no value")
		}

		for i := 0; i < len(specialsTokens); i += 2 {
			matchVar := specialsTokens[i]
			replacementString := specialsTokens[i+1]
			matchRE, err := regexp.Compile(matchVar)
			if err != nil {
				return fmt.Errorf("Could not compile regular expression \"%v\": %v", matchVar, err)
			}
			special := traffic.SpecialPath{
				Match:       matchRE,
				Replacement: replacementString,
			}
			config.Relay.SpecialPaths = append(config.Relay.SpecialPaths, special)
			logger.Printf("Relaying special expression \"%v\" to \"%v\"", special.Match, special.Replacement)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return config, nil
}
