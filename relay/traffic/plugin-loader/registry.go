package plugin_loader

import (
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/content-blocker-plugin"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/cookies-plugin"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/headers-plugin"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/paths-plugin"
	"github.com/fullstorydev/relay-core/relay/plugins/traffic/test-interceptor-plugin"
	"github.com/fullstorydev/relay-core/relay/traffic"
)

// DefaultPlugins is a plugin registry containing all traffic plugins that
// should be available in production. These are the plugins that the relay loads
// on startup.
var DefaultPlugins = []traffic.PluginFactory{
	content_blocker_plugin.Factory,
	cookies_plugin.Factory,
	headers_plugin.Factory,
	paths_plugin.Factory,
}

// TestPlugins is a plugin registry containing test-only traffic plugins. These
// are not loaded by the relay on startup, but can be loaded programmatically in
// tests.
var TestPlugins = []traffic.PluginFactory{
	test_interceptor_plugin.Factory,
}
