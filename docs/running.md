# Running Relay

Pre-built Docker images are available for every version of the Relay so unless you need a custom build that's the way to go:

- [Pre-built Docker images](https://github.com/fullstorydev/relay-core/packages)

## Using Docker images

Pre-built Docker images include all of the default plugins and can be configured for your specific scenario.

### Using the pre-built Docker image

For each release version of Relay there is a Docker image hosted in GitHub's Packages registry. You can see them listed on the [relay-core packages page](https://github.com/fullstorydev/relay-core/packages).

Somewhat annoyingly, GitHub requires authentication in order to use even public Docker images so you'll need to authenticate the Docker CLI using [these instructions](https://help.github.com/en/packages/using-github-packages-with-your-projects-ecosystem/configuring-docker-for-use-with-github-packages#authenticating-to-github-packages).

Once Docker is authenticated with GitHub you can pull the image in the usual way:

	docker pull docker.pkg.github.com/fullstorydev/relay-core/relay-core:v0.1.2

You probably want the latest version so check for a version greater than v0.1.2.

### Building a local Docker image

It can be useful to build a local image for testing or for hosting in your own package registry instead of using one of the [release images](https://github.com/fullstorydev/relay-core/packages) hosted on GitHub's Packages registry.

To create an image:

	cd relay-core/
	docker build -t relay:local-v0 .

### Running the Docker image:

Pre-built:

	docker run -e "RELAY_PORT=8990" -e "RELAY_PLUGINS_PATH=/dist/plugins/" -e "TRAFFIC_RELAY_TARGET=http://127.0.0.1:12346/" --publish 8990:8990 -d docker.pkg.github.com/fullstorydev/relay-core/relay-core:v0.1.2

(update the `v0.1.2` if you're using a different version)

Locally built:

	docker run -e "RELAY_PORT=8990" -e "RELAY_PLUGINS_PATH=/dist/plugins/" -e "TRAFFIC_RELAY_TARGET=http://127.0.0.1:12346/" --publish 8990:8990 -d relay:local-v0

You'll want to change the various environment variables to suite your scenario, as documented in the [example dotenv file](https://github.com/fullstorydev/relay-core/blob/master/config/dotenv.example).

## Using binaries

While we provide [pre-built 'relay' binaries](https://github.com/fullstorydev/relay-core/releases) for each version, to run the Relay you also need a specific directory hierarchy containing [plugins](plugins.md). The default build creates `relay-core/dist/` containing both binary and plugins so the easiest way to get started is:

	cd relay-core/
	make
	# ... build output ...
	cd dist/
	./relay

If you already have a distribution directory with the correct plugins then you can drop in new, pre-built binaries as they're released.

The pre-built Docker images already contain the plugins so if building is annoying then you might try the Docker route.

## Configuration

### Configuration in production

Whether you're using a Docker container or running a binary in a shell script, in production you need to set up environment variables to configure `relay` and its plugins. Recognized environment variables are documented in [`relay-core/config/dotenv.example`](https://github.com/fullstorydev/relay-core/blob/master/config/dotenv.example) and the `relay` command will print helpful information when required variables are missing.

### Configuration during development and local testing

In production you'll set environment variables but during development and local testing it's often easier to use a dotenv file. Relay will look for a dotenv file in the current working directory. You can find an example dotenv file with an explanation for each variable in [`relay-core/config/dotenv.example`](https://github.com/fullstorydev/relay-core/blob/master/config/dotenv.example). Copy that file into `relay-core/dist/.env` and change the values to your desired configuration.
