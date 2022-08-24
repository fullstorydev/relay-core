# Running Relay

Pre-built Docker images are available for every version of the Relay so unless you need a custom build that's the way to go:

- [Pre-built Docker images](https://github.com/fullstorydev/relay-core/packages)

## Using Docker images

Pre-built Docker images include all of the default plugins and can be configured for your specific scenario.

### Using the pre-built Docker image

For each release version of Relay there is a Docker image hosted in GitHub's Packages registry. You can see them listed on the [relay-core packages page](https://github.com/fullstorydev/relay-core/packages).

Somewhat annoyingly, GitHub requires authentication in order to use even public Docker images so you'll need to authenticate the Docker CLI using [these instructions](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry).

Once Docker is authenticated with GitHub you can pull the image in the usual way:

	docker pull docker.pkg.github.com/fullstorydev/relay-core/relay-core:v0.2.0

You probably want the latest version so check for a version greater than v0.2.0.

### Building a local Docker image

It can be useful to build a local image for testing or for hosting in your own package registry instead of using one of the [release images](https://github.com/fullstorydev/relay-core/packages) hosted on GitHub's Packages registry.

To create an image:

	cd relay-core/
	docker build -t relay:local-v0 .

### Running the Docker image:

Pre-built:

	docker run -e "RELAY_PORT=8990" -e "TRAFFIC_RELAY_TARGET=http://127.0.0.1:12346/" --publish 8990:8990 -d docker.pkg.github.com/fullstorydev/relay-core/relay-core:v0.2.0

(update the `v0.2.0` if you're using a different version)

Locally built:

	docker run -e "RELAY_PORT=8990" -e "TRAFFIC_RELAY_TARGET=http://127.0.0.1:12346/" --publish 8990:8990 -d relay:local-v0

You'll want to change the various environment variables to suite your scenario, as documented in the [example dotenv file](https://github.com/fullstorydev/relay-core/blob/master/config/dotenv.example).

## Local Development

To build:

	make

To run tests:

	make test

To run the relay locally after a successful build:

	cd dist
	./relay

## Configuration

### Configuration in production

Whether you're using a Docker container or running a binary in a shell script, in production you need to set up environment variables to configure `relay` and its plugins. Recognized environment variables are documented in [`relay-core/config/dotenv.example`](https://github.com/fullstorydev/relay-core/blob/master/config/dotenv.example) and the `relay` command will print helpful information when required variables are missing.

### Configuration during development and local testing

In production you'll set environment variables but during development and local testing it's often easier to use a dotenv file. Relay will look for a dotenv file in the current working directory. You can find an example dotenv file with an explanation for each variable in [`relay-core/config/dotenv.example`](https://github.com/fullstorydev/relay-core/blob/master/config/dotenv.example). Copy that file into `relay-core/dist/.env` and change the values to your desired configuration.
