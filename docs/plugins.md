# Relay Plugins

The core relay program doesn't include much functionality other than loading plugins and calling into them at appropriate times. Even the main feature of relaying is performed by a plugin!

## Using plugins

If you successfully build using `make` then you'll have a `relay-core/dist/plugins/traffic/` directory holding several .so files. These are the plugin files.

*Load order:*

Plugins are loaded in alpha-numeric sort order, which is why the default build names the plugins like so:

- 010-relay.so
- 020-monitor.so
- 030-logging.so

The order of traffic plugins matters because each plugin is given the opportunity to service incoming requests. The default order lets the relay plugin attempt to relay the request and then the monitor and logging plugins can do their work assuming that the request is handled.

If you write a plugin then you'll want to name it in such a way that it is loaded in the order that you expect. For example, if you want your plugin to load between the relay and monitor plugins then you'd name it `015-something.so` and because 015 is between 010 and 020 it'll be loaded as expected.

## Writing plugins

Relay plugins are implemented as [go plugins](https://github.com/golang/go/wiki/Modules) and that comes with a few tricky bits. The most common error is for a plugin to not expose exactly the expected interface and then fail to load.

There are three traffic plugins (currently the only plugin type) in the `relay-core` source code that you can use as example code for your plugin:
- [relay](https://github.com/fullstorydev/relay-core/tree/master/go/src/relay/plugins/traffic/relay/main)
- [monitor](https://github.com/fullstorydev/relay-core/tree/master/go/src/relay/plugins/traffic/monitor/main)
- [logging](https://github.com/fullstorydev/relay-core/tree/master/go/src/relay/plugins/traffic/logging/main)

To build a plugin you need to use `go build -buildmode=plugin ...` which you can see used in the `plugins` target of Relay's [Makefile](https://github.com/fullstorydev/relay-core/blob/master/Makefile).

