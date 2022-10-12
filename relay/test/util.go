package test

import (
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/traffic"
	"github.com/fullstorydev/relay-core/relay/traffic/plugin-loader"
)

// WithCatcherAndRelay is a helper function that wraps the setup and teardown
// required by most relay unit tests. Given a set of plugin configuration
// variables and an allowlist of plugins to load, it loads and configures the
// plugins, starts the catcher and relay services, and invokes the provided
// action function, which should handle the actual testing. Afterwards, it
// ensures that everything gets torn down so that the next test can start from a
// clean slate.
func WithCatcherAndRelay(
	t *testing.T,
	envVars map[string]string,
	pluginFactories []traffic.PluginFactory,
	action func(catcherService *catcher.Service, relayService *relay.Service),
) {
	catcherService := catcher.NewService()
	if err := catcherService.Start("localhost", 0); err != nil {
		t.Errorf("Error starting catcher: %v", err)
		return
	}
	defer catcherService.Close()

	if envVars == nil {
		envVars = map[string]string{}
	}
	envVars["RELAY_PORT"] = "0"
	envVars["TRAFFIC_RELAY_TARGET"] = catcherService.HttpUrl()

	relayService, err := setupRelay(envVars, pluginFactories)
	if err != nil {
		t.Errorf("Error setting up relay: %v", err)
		return
	}

	if err := relayService.Start("localhost", 0); err != nil {
		t.Errorf("Error starting relay: %v", err)
		return
	}
	defer relayService.Close()

	action(catcherService, relayService)
}

func setupRelay(
	envVars map[string]string,
	pluginFactories []traffic.PluginFactory,
) (*relay.Service, error) {
	envProvider := commands.NewTestEnvironmentProvider(envVars)
	env := commands.NewEnvironment(envProvider)
	config, err := relay.ReadConfig(env)
	if err != nil {
		return nil, err
	}

	trafficPlugins, err := plugin_loader.Load(pluginFactories, env)
	if err != nil {
		return nil, err
	}

	return relay.NewService(config.Relay, trafficPlugins), nil
}
