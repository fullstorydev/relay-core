# Relay

*This project is not ready for use in production.*

If you're interested, the initial work is happening in the `development` branch.

This project is covered by the MIT License. See [LICENSE](LICENSE) for details.


## Docker image

Create an image:

	docker build -t relay:v0 .

Run the relay:

	docker run -e "RELAY_PORT=8990" -e "RELAY_PLUGINS_PATH=/dist/plugins/" -e "TRAFFIC_RELAY_TARGET=https://www.wikipedia.org/" --publish 8990:8990 -d relay:v0