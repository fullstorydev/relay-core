package test

import (
	"testing"

	"github.com/fullstorydev/relay-core/catcher"
	"github.com/fullstorydev/relay-core/relay"
	"github.com/fullstorydev/relay-core/relay/commands"
	"github.com/fullstorydev/relay-core/relay/plugins"
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
	pluginAllowlist map[string]bool,
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
	relayService, err := setupRelay(env, pluginAllowlist)
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

func setupRelay(env commands.Environment, pluginAllowlist map[string]bool) (*relay.Service, error) {
	var pluginsPath string = "../../../../dist/plugins"

	if pluginAllowlist == nil {
		pluginAllowlist = make(map[string]bool)
	}

	// Always allowlist the Relay and Logging plugins, since in practice every
	// test wants them.
	pluginAllowlist[relay_plugin.Factory.Name()] = true
	pluginAllowlist[logging_plugin.Factory.Name()] = true

	plugs := plugins.New()
	if err := plugs.LoadWithFilter(pluginsPath, pluginAllowlist); err != nil {
		return nil, err
	}
	if err := plugs.ConfigurePlugins(env); err != nil {
		return nil, err
	}

	return relay.NewService(plugs), nil
}
