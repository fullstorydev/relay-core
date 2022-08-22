# Relay Plugins

The core relay program's functionality can be extended using plugins. Much of the built-in functionality of the relay is implemented using these plugins, in fact. Plugins are built into the relay binary at compile time, so you don't need to do anything special to load them.

## Writing plugins

To write a plugin for the relay, you need to write implementations for the [PluginFactory and Plugin interfaces](https://github.com/fullstorydev/relay-core/blob/master/relay/traffic/plugin-interfaces.go). The [built-in plugins](https://github.com/fullstorydev/relay-core/tree/master/relay/plugins/traffic) may serve as a useful starting point.

Plugins are built and tested as part of the relay code, so you can simply run `make` to build your plugin or `make test` to run its tests.

To use your new plugin outside of tests, you'll also need to add it to the [DefaultPlugins list](https://github.com/fullstorydev/relay-core/blob/master/relay/main/main.go). This will make the `relay` program load the plugin at startup.
