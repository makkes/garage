# Garage

[![OCI distribution 1.0 conformance](https://github.com/makkes/garage/actions/workflows/conformance.yaml/badge.svg)](https://github.com/makkes/garage/actions/workflows/conformance.yaml)

A prototypical OCI registry that's 100% spec-conformant.

## Getting started

The registry can be run from source, from pre-built binaries or from container images.

### Running from source

You'll need to have a version of `make` and `go` installed. Then run:

```sh
make run
```

### Using pre-built binaries

Download the binary for your architecture from the [Releases](https://github.com/makkes/garage/releases) page and run it.

### Using a container image

To run the image in Docker, run this command (replace `VERSION` with the actual version you'd like to run):

```sh
docker run -d --name garage -p 8080:8080 ghcr.io/makkes/garage:VERSION
```

Keep in mind that any data stored in the registry only persists until the container is deleted. To make it persist, use a Docker volume:

```sh
docker volume create garage_data
docker run -d --name garage -p 8080:8080 -v garage_data:/data ghcr.io/makkes/garage:VERSION
```

## Configuration

Garage can be configured through a configuration file, command-line arguments or environment variables. A sample configuration file is provided in [config.yaml](./config.yaml).
