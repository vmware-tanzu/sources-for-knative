# E2E tests
For insight into how the tests are run, you can check out `.github/workflows/kind-e2e.yaml`.

## Setup

Before you begin, make sure you have your `KO_DOCKER_REPO` env var set to whatever registry you like to use.

To run the test, you'll need to upload the test images to a registry. This can be done by running `upload-test-images.sh`.

This document assumes you have already installed the Tanzu Sources, as described [here](/README.md)

#### vCenter Simulator Image

By default, the tests use the vCenter Simulator image `vmware/vcsim:latest`, which is hosted on Dockerhub.

If you wish to use a `vcsim` image hosted on a different registry, you can build and store it yourself with [`ko`](https://github.com/google/ko), like this:
```
export VCSIM_IMAGE=$(ko publish -B github.com/vmware/govmomi/vcsim)
```

## Running the tests
The tests can be run with `go test`, like this:
```
go test -v -race -count=1 -tags=e2e ./test/e2e
```

You can run a specific test with the `-run` flag, like this:
```
go test -v -race -count=1 -tags=e2e -run='^(TestSource)$' ./test/e2e
```
