# VMware Tanzu Sources for Knative

[![GoDoc](https://godoc.org/github.com/vmware-tanzu/sources-for-knative?status.svg)](https://godoc.org/github.com/vmware-tanzu/sources-for-knative)
[![Go Report Card](https://goreportcard.com/badge/vmware-tanzu/sources-for-knative)](https://goreportcard.com/report/vmware-tanzu/sources-for-knative)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://knative.slack.com)
[![codecov](https://codecov.io/gh/vmware-tanzu/sources-for-knative/branch/master/graph/badge.svg?token=QwWjUwiLIN)](undefined)

This repo is the home for VMware-related event sources compatible with the
[Knative](https://knative.dev) project.


This repo is under active development to get a Knative compatible Event `Source`
for VMware events, e.g. VMware vSphere incl. a `Binding` to easily access the
vSphere API from Kubernetes objects, e.g. a `Job`.

⚠️ **NOTE:** To run these examples, you will need
[ko](https://github.com/google/ko) installed or use a
[release](https://github.com/vmware-tanzu/sources-for-knative/releases)
(preferred) and deploy it via `kubectl`.

## Available `Sources` and `Bindings`

- `VSphereSource` to create VMware vSphere (vCenter) event sources
- `VSphereBinding` to inject VMware vSphere (vCenter) credentials

## Install Tanzu Sources for Knative

### Install via Release (`latest`)

```
kubectl apply -f https://github.com/vmware-tanzu/sources-for-knative/releases/latest/download/release.yaml
```

### Install from Source

Install the CRD providing the control / dataplane for the various `Sources` and
`Bindings`:

```shell
# define environment variables accordingly, e.g. when using kind
# export KIND_CLUSTER_NAME=horizon
# export KO_DOCKER_REPO=kind.local

ko apply -BRf config
```

## Examples

To see examples of the Source and Binding in action, check out our
[samples](./samples/README.md) directory.

## Basic `VSphereSource` Example

The `VSphereSource` provides a simple mechanism to enable users to react to
vSphere events.

In order to receive events from vSphere (i.e. vCenter) these are the **key
parts** in the configuration:

1. The vCenter address and secret information.
1. Where to send the events.
1. Checkpoint behavior.
1. Payload encoding scheme

```yaml
apiVersion: sources.tanzu.vmware.com/v1alpha1
kind: VSphereSource
metadata:
  name: source
spec:
  # Where to fetch the events, and how to auth.
  address: https://vcenter.corp.local
  skipTLSVerify: true
  secretRef:
    name: vsphere-credentials

  # Where to send the events.
  sink:
    uri: http://where.to.send.stuff

  # Adjust checkpointing and event replay behavior
  checkpointConfig:
    maxAgeSeconds: 300
    periodSeconds: 10

  # Set the CloudEvent data encoding scheme to JSON
  payloadEncoding: application/json
```

Let's walk through each of these.

### Authenticating with vSphere

Let's focus on this part of the sample source:

```yaml
# Where to fetch the events, and how to auth.
address: https://vcenter.corp.local
skipTLSVerify: true
secretRef:
  name: vsphere-credentials
```

- `address` is the URL of ESXi or vCenter instance to connect to (same as
  `VC_URL`).
- `skipTLSVerify` disables certificate verification (same as `VC_INSECURE`).
- `secretRef` holds the name of the Kubernetes secret with the following form:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: vsphere-credentials
type: kubernetes.io/basic-auth
stringData:
  # Same as VC_USERNAME
  username: ...
  # Same as VC_PASSWORD
  password: ...
```

### Delivering Events

Let's focus on this part of the sample source:

```yaml
# Where to send the events.
sink:
  uri: http://where.to.send.stuff
```

The simplest way to deliver events is to simply send them to an accessible
endpoint as above, but there are a few additional options to explore.

To deliver events to a Kubernetes Service in the same namespace you can use:

```yaml
# Where to send the events.
sink:
  ref:
    apiVersion: v1
    kind: Service
    name: the-service-name
```

To deliver events to a [Knative Service](https://knative.dev/docs/serving)
(scales to zero) in the same namespace you can use:

```yaml
# Where to send the events.
sink:
  ref:
    apiVersion: serving.knative.dev/v1
    kind: Service
    name: the-knative-service-name
```

To deliver events to a [Knative Broker](https://knative.dev/docs/eventing) in
the same namespace (e.g. here the `default`) you can use:

```yaml
# Where to send the events.
sink:
  ref:
    apiVersion: eventing.knative.dev/v1beta1
    kind: Broker
    name: default
```

### Configuring Checkpoint and Event Replay

Let's focus on this section of the sample source:

```yaml
# Adjust checkpointing and event replay behavior
checkpointConfig:
  maxAgeSeconds: 300
  periodSeconds: 10
```

The `Source` controller will periodically checkpoint its progress in the vCenter
event stream ("history") by using a Kubernetes `ConfigMap` as storage backend.
The name of the `ConfigMap` is `<name_of_source>-configmap` (see example below).
By default, checkpoints will be created every `10 seconds`. The minimum
checkpoint frequency is `1s` but be aware of potential load on the Kubernetes
API this might cause.

Upon start, the controller will look for an existing checkpoint (`ConfigMap`).
If a valid one is found, and if it is within the history replay window
(`maxAgeSeconds`, see below), it will start replaying the vCenter event stream
from the timestamp specified in the checkpoint's `"lastEventKeyTimestamp"` key.
If the timestamp is older than `maxAgeSeconds`, the controller will start
replaying from `maxAgeSeconds` before the current vCenter time (UTC).  If there
is no existing checkpoint, the controller will not replay any events, regardless
of what value is in `maxAgeSeconds`. This is to ensure that when upgrading from
an earlier version of the controller where checkpointing was not implemented, no
events will be accidentally replayed.

Checkpointing is useful to guarantee **at-least-once** event delivery semantics,
e.g. to guard against lost events due to controller downtime (maintenance,
crash, etc.). To influence the checkpointing logic, these parameters are
available in `spec.checkpointConfig`:

- `periodSeconds`:
  - Description: how often to save a checkpoint (**RPO**, recovery point
    objective)
  - Minimum: `1`
  - Default (when `0` or unspecified): `10`
- `maxAgeSeconds`:
  - Description: the history window when replaying the event history (**RTO**,
    recovery time objective)
  - Minimum: `0` (disables event replay, see below)
  - Default: `300`

⚠️ **IMPORTANT:** Checkpointing itself cannot be disabled and there will be
exactly zero or one checkpoint per controller. If **at-most-once** event
delivery is desired, i.e. no event replay upon controller start, simply set
`maxAgeSeconds: 0`.

To reduce load on the Kubernetes API, a new checkpoint will not be saved under
the following conditions:

- When **all** events in a batch could not be sent to the `sink` (note: partial
  success, i.e. successful events in a batch until the first failed event will
  be checkpointed though)
- When the vCenter event polling logic does not return any new events (note:
  fixed backoff logic is applied to reduce load on vCenter)

⚠️ **IMPORTANT:** When a `VSphereSource` is deleted, the corresponding
checkpoint (`ConfigMap`) will also be **deleted**! Make sure to backup any
checkpoint before deleting the `VSphereSource` if this is required for
auditing/compliance reasons.

Here is an example of a JSON-encoded checkpoint for a `VSphereSource` named
`vc-source`:

```bash
kubectl get cm vc-source-configmap -o jsonpath='{.data}'

# output edited for better readability
{
  "checkpoint": {
    "vCenter": "10.161.153.226",
    "lastEventKey": 17208,
    "lastEventType": "UserLogoutSessionEvent",
    "lastEventKeyTimestamp": "2021-02-15T19:20:35.598999Z",
    "createdTimestamp": "2021-02-15T19:20:36.3326551Z"
  }
}
```

### Configuring CloudEvent Payload Encoding

Let's focus on this section of the sample source:

```yaml
# Set the CloudEvent data encoding scheme to JSON
payloadEncoding: application/json
```

The default CloudEvent payload encoding scheme, i.e.
[`datacontenttype`](https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#datacontenttype),
produced by a `VSphereSource` in the `v1alpha1` API is `application/xml`.
Alternatively, this can be changed to `application/json` as shown in the sample
above. Other encoding schemes are currently **not implemented**.

## Basic `VSphereBinding` Example

The `VSphereBinding` provides a simple mechanism for a user application to call
into the vSphere API. In your application code, simply write:

```go
import "github.com/vmware-tanzu/sources-for-knative/pkg/vsphere"

// This returns a github.com/vmware/govmomi.Client
client, err := New(ctx)
if err != nil {
	log.Fatalf("Unable to create vSphere client: %v", err)
}

// This returns a github.com/vmware/govmomi/vapi/rest.Client
restclient, err := New(ctx)
if err != nil {
	log.Fatalf("Unable to create vSphere REST client: %v", err)
}
```

This will authenticate against the bound vSphere environment with the bound
credentials. This same code can be moved to other environments and bound to
different vSphere endpoints without being rebuilt or modified!

Now let's take a look at how `VSphereBinding` makes this possible.

In order to bind an application to a vSphere endpoint, there are two key parts:

1. The vCenter address and secret information (identical to Source above!)
2. The application that is being bound (aka the "subject").

```yaml
apiVersion: sources.tanzu.vmware.com/v1alpha1
kind: VSphereBinding
metadata:
  name: binding
spec:
  # Where to fetch the events, and how to auth.
  address: https://my-vsphere-endpoint.local
  skipTLSVerify: true
  secretRef:
    name: vsphere-credentials

  # Where to bind the endpoint and credential data.
  subject:
    apiVersion: apps/v1
    kind: Deployment
    name: my-simple-app
```

Authentication is identical to source, so let's take a deeper look at subjects.

### Binding applications (aka subjects)

Let's focus on this part of the sample binding:

```yaml
# Where to bind the endpoint and credential data.
subject:
  apiVersion: apps/v1
  kind: Deployment
  name: my-simple-app
```

In this simple example, the binding is going to inject several environment
variables and secret volumes into the containers in this exact Deployment
resource.

If you would like to target a _selection_ of resources you can also write:

```yaml
# Where to bind the endpoint and credential data.
subject:
  apiVersion: batch/v1
  kind: Job
  selector:
    matchLabels:
      foo: bar
```

Here the binding will apply to every `Job` in the same namespace labeled
`foo: bar`, so this can be used to bind every `Job` stamped out by a `CronJob`
resource.

At this point, you might be wondering: what kinds of resources does this
support? We support binding all resources that embed a Kubernetes PodSpec in the
following way (standard Kubernetes shape):

```yaml
spec:
  template:
    spec: # This is a Kubernetes PodSpec.
      containers:
      - image: ...
      ...
```

This has been tested with:

- Knative `Service` and `Configuration`
- `Deployment`
- `Job`
- `DaemonSet`
- `StatefulSet`

## Changing Log Levels

All components follow Knative logging convention and use the
[zap](https://pkg.go.dev/go.uber.org/zap) structured logging library. The log
level (`debug`, `info`, `error`,
[etc.](https://github.com/uber-go/zap/blob/6f34060764b5ea1367eecda380ba8a9a0de3f0e6/zapcore/level.go#L138))
is configurable per component, e.g. `vsphere-source-webhook`, `VSphereSource`
adapter, etc.

The default logging level is `info`. The following sections describe how to
change the individual component log levels.

### `Source` Adapter Log Level

The log level for adapters, e.g. a particular `VSphereSource` `deployment` can
be changed at runtime via the `config-logging` `ConfigMap` which is
[created](./config/config-logging.yaml) when deploying the Tanzu Sources for
Knative manifests in this repository.

⚠️ **Note:** These settings will affect **all adapter** deployments. Changes to
a particular adapter deployment are currently not possible.

```
kubectl -n vmware-sources edit cm config-logging
```

An interactive editor opens. Change the settings in the JSON object under the
`zap-logger-config` key. For example, to change the log level from `info` to
`debug` use this configuration in the editor:

```yaml
apiVersion: v1
data:
  # details omitted
  zap-logger-config: |
    {
      "level": "debug"
      "development": false,
      "outputPaths": ["stdout"],
      "errorOutputPaths": ["stderr"],
      "encoding": "json",
      "encoderConfig": {
        "timeKey": "ts",
        "levelKey": "level",
        "nameKey": "logger",
        "callerKey": "caller",
        "messageKey": "msg",
        "stacktraceKey": "stacktrace",
        "lineEnding": "",
        "levelEncoder": "",
        "timeEncoder": "iso8601",
        "durationEncoder": "",
        "callerEncoder": ""
      }
    }
```

Save and leave the interactive editor to apply the `ConfigMap` changes.
Kubernetes will validate and confirm the changes:

```
configmap/config-logging edited
```

To verify that the `Source` adapter owners (e.g. `vsphere-source-webhook` for a
`VSphereSource`) have noticed the desired change, inspect the log messages of
the owner (here: `vsphere-source-webhook`) `Pod`:

```
vsphere-source-webhook-f7d8ffbc9-4xfwl vsphere-source-webhook {"level":"info","ts":"2022-03-29T12:25:20.622Z","logger":"vsphere-source-webhook","caller":"vspheresource/vsphere.go:250","msg":"update from logging ConfigMap{snip...}
```

⚠️ **Note:** To avoid unwanted disruption during event retrieval/delivery, the
changes are **not applied** automatically to deployed adapters, i.e.
`VSphereSource` adapter, etc. The operator is in full control over the lifecycle
(downtime) of the affected `Deployment(s)`.

To make the changes take affect for existing adapter `Deployment`, an operator
needs to manually perform a rolling upgrade. The existing adapter `Pod` will be
terminated and a new instanced created with the desired log level changes.

```
kubectl get vspheresource
NAME                SOURCE                     SINK                                                                              READY   REASON
example-vc-source   https://my-vc.corp.local   http://broker-ingress.knative-eventing.svc.cluster.local/default/example-broker   True

kubectl rollout restart deployment/example-vc-source-deployment
deployment.apps/example-vc-source-deployment restarted
```

⚠️ **Note:** To avoid losing events due to this (brief) downtime, consider
enabling the [Checkpointing](#configuring-checkpoint-and-event-replay)
capability.

### `Controller` and `Webhook` Log Level

Each of the available Tanzu Sources for Knative is backed by at least a
particular `webhook`, e.g. `vsphere-source-webhook` and optionally a
`controller`. 

💡 **Note:** The `VSphereSource` and `VSphereBinding` implementation uses a
combined [deployment](./config/vsphere/webhook.yaml) `vsphere-source-webhook`
for the `webhook` and `controller`

Just like with the adapter `Deployment`, the log level for `webhook` and
`controller` instances can be changed at runtime.

```
kubectl -n vmware-sources edit cm config-logging
```

An interactive editor opens. Change the corresponding component log level under
the respective component `key`. The following example shows how to change the
log level for the combined `VSphereSource` `webhook` and `controller` component
from `info` to `debug`:

```yaml
apiVersion: v1
data:
  loglevel.controller: info # generic knative settings, ignore this
  loglevel.webhook: info # generic knative settings, ignore this
  loglevel.vsphere-source-webhook: debug # <- changed from info to debug
  zap-logger-config: |
    {
      "level": "info",
      "development": false,
      "outputPaths": ["stdout"],
      "errorOutputPaths": ["stderr"],
    # details omitted
```

Save and leave the interactive editor to apply the `ConfigMap` changes.
Kubernetes will validate and confirm the changes:

```
configmap/config-logging edited
```

To verify the changes, inspect the log messages of the `vsphere-source-webhook`
`Pod`:

```
vsphere-source-webhook-f7d8ffbc9-4xfwl vsphere-source-webhook {"level":"info","ts":"2022-03-29T12:22:25.630Z","logger":"vsphere-source-webhook","caller":"logging/config.go:209","msg":"Updating logging level for vsphere-source-webhook from info to debug.","commit":"26d67c5"}
```
