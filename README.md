# SHIP - SprintHive Innovation Platform
Install a plug-and-play microservices environment into your Kubernetes cluster.

There are a many infrastructure components needed to run an effective microservices architecture. Researching, configuring and integrating all these components takes a great deal of time and effort, effort that could be better spent working on the services that are core to your business. This project aims to create a fully functioning microservices environment on an existing Kubernetes cluster within minutes by installing a collection of open-source tools that have been configured to work seamlessly with each other.

Components that will be installed currently include:
* Ingress GW (Kong)
* Ingress GW Database (Cassandra)
* Logging database (Elasticsearch)
* Log collector (Fluent-bit)
* Tracing (Zipkin)
* Metric database (Prometheus)
* Metric Visualization (Grafana)
* Log Viewer (Kibana)
* CI/CD (Jenkins)
* Artifact repository (Nexus)

## Build requirements
### Easy Mode
* Docker (the build script uses Docker to create the binary)

### Non-Docker
See Dockerfile for requirements

## Building
### Easy Mode
Execute: `./build.sh`

### Non-Docker
See the contents of build.sh for what needs to be done.

## Runtime requirements
* [Docker](https://docker.com) used to create a go environment to build the ship binary
* [Kubernetes](https://github.com/kubernetes/kubernetes) cluster in which the components will be installed
* [Kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) CLI tool in your `$PATH`
* [Helm](https://github.com/kubernetes/helm) CLI tool in your `$PATH` and corresponding Helm Tiller installed in the Kubernetes cluster (see Helm page for details)

## Usage
### Install
Execute: `./ship install`

## More info

For a more detailed guide have a look at the [wiki](https://github.com/SprintHive/ship/wiki)
