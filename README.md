# Relay

The Relay project provides a service that relays HTTP requests (including WebSockets) to a different service. The most common use is to relay requests from a hostname in one domain to a service in a different domain.

For example, you could run an instance of Relay at `design-tool.your-domain.com` and configured it to relay requests to a third party service like `design-tool.com`.

Relay gives you more control and monitoring possibilities over network traffic from your users' browsers that would normally go directly to a third party service.

To get started, check out the [Running Relay](./docs/running.md) document.

This project is covered by the MIT License. See [LICENSE](LICENSE) for details.
