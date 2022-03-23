## Binary names

The binaries in this directory are grouped by and prefixed with the
corresponding VMware product name, e.g. `vsphere` or `horizon`.

When published via `KO_DOCKER_REPO=<registry>/vmware ko apply -BRf config` the
resulting images are named `<registry>/vmware/<product>-{adapter|controller}`,
e.g. `docker.io/vmware/vsphere-adapter`
