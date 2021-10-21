# VMware Tanzu Sources for Knative

This repo will be the home for VMware-related event sources compatible with the
[Knative](https://knative.dev) project.


[![GoDoc](https://godoc.org/github.com/vmware-tanzu/sources-for-knative?status.svg)](https://godoc.org/github.com/vmware-tanzu/sources-for-knative)
[![Go Report Card](https://goreportcard.com/badge/vmware-tanzu/sources-for-knative)](https://goreportcard.com/report/vmware-tanzu/sources-for-knative)
[![Slack Status](https://img.shields.io/badge/slack-join_chat-white.svg?logo=slack&style=social)](https://knative.slack.com)
[![codecov](https://codecov.io/gh/vmware-tanzu/sources-for-knative/branch/master/graph/badge.svg?token=QwWjUwiLIN)](undefined)

This repo is under active development to get a Knative compatible Event Source
for vSphere events, and a Binding to easily access the VSphere API.

⚠️ **NOTE:** To run these examples, you will need
[ko](https://github.com/google/ko) installed or use a
[release](https://github.com/vmware-tanzu/sources-for-knative/releases) and
deploy it via `kubectl`.

## Install Tanzu Sources for Knative

### Install via Release

```
kubectl apply -f https://github.com/vmware-tanzu/sources-for-knative/releases/download/v0.21.0/release.yaml
```

### Install from Source

Install the CRD providing the control / dataplane for the
`VSphere{Source,Binding}`:

```shell
ko apply -f config
```

## Samples

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

Upon start, the controller will look for an existing checkpoint (`ConfigMap`)
and, if a valid one is found, use its last event (timestamp) to start replaying
the vCenter history stream from that point in time. The default history replay
window is `5 minutes`, i.e. events within this time window will be replayed and
sent to the `sink`. By default, checkpoints will be created every `10 seconds`.
The minimum checkpoint frequency is `1s` but be aware of potential load on the
Kubernetes API this might cause.

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
  - Default: `n/a` (must be explicitly specified)

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
