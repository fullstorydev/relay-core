package test

import (
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/plugins"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/logging-plugin"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/relay-plugin"
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
	env commands.Environment,
	pluginFactories []traffic.PluginFactory,
	action func(catcherService *catcher.Service, relayService *relay.Service),
) {
	if env == nil {
		env = commands.Environment{}
	}

	catcherService := catcher.NewService()
	if err := catcherService.Start("localhost", 0); err != nil {
		t.Errorf("Error starting catcher: %v", err)
		return
	}
	defer catcherService.Close()

	env["TRAFFIC_RELAY_TARGET"] = catcherService.HttpUrl()
	relayService, err := setupRelay(env, pluginFactories)
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

func setupRelay(env commands.Environment, pluginFactories []traffic.PluginFactory) (*relay.Service, error) {
	// Always include the Relay and Logging plugins, since in practice every
	// test wants them.
	pluginFactories = append(pluginFactories, relay_plugin.Factory)
	pluginFactories = append(pluginFactories, logging_plugin.Factory)

	envProvider := commands.NewTestEnvironmentProvider(env)
	trafficPlugins, err := plugins.Load(pluginFactories, envProvider)
	if err != nil {
		return nil, err
	}

	return relay.NewService(trafficPlugins), nil
}
