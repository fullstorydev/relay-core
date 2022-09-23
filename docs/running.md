# Running Relay

The Relay service is available as a Docker image. Unless you need a custom
build, using a Docker image is the easiest way to get up and running quickly.

- [Pre-built Docker images](https://github.com/fullstorydev/relay-core/packages)

## Getting Docker images

In most cases, using a pre-built Docker image is the preferred approach to using
Relay. If you're working on the Relay code or you need a custom build, it can
also be useful to build a local Docker image. Both approaches are explained
below.

### Pulling a pre-built Docker image

GitHub requires authentication in order to use even public Docker images, so
you'll need to authenticate the Docker CLI using
[these instructions](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry#authenticating-to-the-container-registry).

Once Docker is authenticated with GitHub you can pull the image using `docker pull`.
You can find the necessary command on the
[relay-core package page](https://github.com/fullstorydev/relay-core/pkgs/container/relay-core%2Frelay-core).

### Building a local Docker image

If you're working on the Relay code or you need a custom build, you may find it
useful to build a Docker image locally. You can do as follows:

	cd relay-core/
	docker build -t relay:local-v0 .

## Running the Relay using a Docker image

Once you've followed the directions in the previous section and obtained a Relay
Docker image, you're ready to run Relay. The examples below use the image
name `relay:image`; you'll need to substitute in the name of the image you
obtained above. For example, if you built a local Docker image, you'll want to
replace `relay:image` with `relay:local-v0`.

Running Relay requires that you set one required option: the target. This
option tells Relay where to direct the traffic it receives. It's specified
as a URL, but you only need to provide the protocol and host; the path and query
parameters are derived from incoming requests. As a convenience, you can specify
the target by starting the container with the `TRAFFIC_RELAY_TARGET` environment
variable set appropriately:

	docker run -e "TRAFFIC_RELAY_TARGET=https://target.example:12346" --publish 8990:8990 -it --rm relay:image

You can also set the target using the configuration file. The configuration file
gives you access to quite a few more advanced features; for more details, see
the comments in the
[default configuration file](https://github.com/fullstorydev/relay-core/blob/master/relay.yaml).
You can use this file as a starting point when configuring Relay. You can test
your configuration file by piping it into the Docker container when you run it.
The `--config -` argument tells Relay to read its configuration from stdin:

	cat custom-relay.yaml | docker run --publish 8990:8990 -i --rm relay:image --config -

In production you may find it more convenient to bind mount your custom
configuration file over the default one, which is located at
`/etc/relay/relay.yaml`, or to mount a volume containing the configuration file
and use the `--config` option to tell Relay where to find it. For more details
on these mechanisms, consult the Docker documentation. If you're hosting the
image yourself, you also have the option of updating `relay.yaml` and rebuilding
the Docker image, so that your desired configuration becomes the new default.

The configuration file can reference environment variables. You may find this
useful in more complex scenarios. The
[default configuration file](https://github.com/fullstorydev/relay-core/blob/master/relay.yaml)
references a number of environment variables; you can use these variables as an
alternative way to configure Relay. The `TRAFFIC_RELAY_TARGET` variable
discussed above is an example of using this kind of environment variable
reference.

## Local development

Working on the Relay code is easy; the only dependencies are the standard Unix
development tools and a working installation of Go.

To build:

	make

To run tests:

	make test

To run Relay locally after a successful build:

	./dist/relay

By default, Relay reads its configuration from `relay.yaml` in the current
working directory. You can explicitly specify the path to the configuration file
using the `--config` option:

	./dist/relay --config /etc/relay/relay.yaml

If you plan to add new functionality to Relay, it's important to understand
its plugin-based architecture; you can read more about that [here](plugins.md].
