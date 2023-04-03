# Relay Plugins

The core relay program's functionality can be extended using plugins. Much of
the built-in functionality of Relay is implemented using these plugins, in
fact. Plugins are built into the `relay` binary at compile time, so you don't need
to do anything special to load them.

## Writing plugins

To write a plugin for Relay, you need to write implementations for the
[PluginFactory and Plugin interfaces](https://github.com/fullstorydev/relay-core/blob/master/relay/traffic/plugin-interfaces.go).
The [built-in plugins](https://github.com/fullstorydev/relay-core/tree/master/relay/plugins/traffic)
may serve as a useful starting point.

Plugins are built and tested as part of the Relay code, so you can simply run
`make` to build your plugin or `make test` to run its tests.

To use your new plugin, you'll need to add it to the either the `DefaultPlugins`
or `TestPlugins`
[registry](https://github.com/fullstorydev/relay-core/blob/master/relay/traffic/plugin-loader/registry.go).
Plugins in the `DefaultPlugins` registry are loaded by `relay` program at
startup. Plugins in the `TestPlugins` registry are not loaded by the `relay`
program, but are available in unit tests.
