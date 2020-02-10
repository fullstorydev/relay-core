# Running Relay

To run the Relay you need the `relay` binary and you need a specific directory hierarchy containing [plugins](plugins.md). The default build creates `relay-core/dist/` containing both so the easiest way to get started is:

	cd relay-core/
	make
	# ... build output ...
	cd dist/
	./relay

If you do that then you'll see error messages about missing environment variables because the `relay` and the plugins need a bit of configuration.

## Configuration during development and local testing

While in production you'll set environment variables, during development and local testing it's often easier to use a dotenv file. Relay will look for a dotenv file in the current working directory. You can find an example dotenv file in `relay-core/config/dotenv.example`. Copy that file into `relay-core/dist/.env` and change the values to your desired configuration.

## Configuration in production

Whether you're using a Docker container or running the binary in a shell script, in production you need to set up environment variables to configure `relay` and its plugins. Recognized environment variables are documented in `relay-core/config/dotenv.example` and the `relay` command will print helpful information when required variables are missing.

## Using the per-release Docker image

For each release version of Relay there is a Docker image hosted in GitHub's Packages registry. You can see them listed on the [relay-core packages page](https://github.com/fullstorydev/relay-core/packages).

Somewhat annoyingly, GitHub requires authentication in order to use even public Docker images so you'll need to authenticate the Docker CLI using these instructions. Once that's complete, you can pull the image in the usual way:

	docker pull docker.pkg.github.com/fullstorydev/relay-core/relay-core:v0.0.1-alpha3

For production use you'll want to choose the latest, non-alpha version.

## Building a local Docker image

It can useful to build a local image for testing or for hosting in your own package registry instead of using one of the [release images](https://github.com/fullstorydev/relay-core/packages) hosted on GitHub's Packages registry.

To create an image:

	cd relay-core/
	docker build -t relay:local-v0 .

To run the image:

	docker run -e "RELAY_PORT=8990" -e "RELAY_PLUGINS_PATH=/dist/plugins/" -e "TRAFFIC_RELAY_TARGET=http://127.0.0.1:12346/" --publish 8990:8990 -d relay:local-v0
