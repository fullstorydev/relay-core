package relay

import (
	"fmt"
	"net/url"

	"github.com/fullstorydev/relay-core/relay/config"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

type Options struct {
	Service *ServiceOptions
	Relay   *traffic.RelayOptions
}

func ReadOptions(configFile *config.File) (*Options, error) {
	options := &Options{
		Service: NewDefaultServiceOptions(),
		Relay:   traffic.NewDefaultRelayOptions(),
	}

	configSection, err := configFile.LookupRequiredSection("relay")
	if err != nil {
		return nil, err
	}

	if port, err := config.LookupRequired[int](configSection, "port"); err != nil {
		return nil, err
	} else {
		logger.Printf("Port: %v\n", port)
		options.Service.Port = port
	}

	if err := config.ParseRequired(configSection, "target", func(key, value string) error {
		logger.Printf("Target: %v\n", value)
		if targetURL, err := url.Parse(value); err != nil {
			return err
		} else if targetURL.Scheme == "" || targetURL.Host == "" {
			return fmt.Errorf("Invalid or relative target URL")
		} else {
			options.Relay.TargetScheme = targetURL.Scheme
			options.Relay.TargetHost = targetURL.Host
			return nil
		}
	}); err != nil {
		return nil, err
	}

	if maxBodySize, err := config.LookupOptional[int64](configSection, "max-body-size"); err != nil {
		return nil, err
	} else if maxBodySize != nil {
		logger.Printf("Maximum response body size: %v\n", *maxBodySize)
		options.Relay.MaxBodySize = *maxBodySize
	}

	return options, nil
}
