## `HorizonSource` Example

To create a `HorizonSource` the Horizon server API version must be at least
`2106`. There are over 850+ events that are available and for a complete list,
please take a look [here](https://github.com/lamw/horizon-event-mapping/).


### Configure the Source

Modify the `horizon-source.yml` according to your environment.

`address` is the HTTPs endpoint of the Horizon API server. To skip TLS and
certificate verification, set `skipTLSVerify` to `true`.

Change the values under `sink` to match your Knative Eventing environment.

If the specified `serviceAccountName` does not exist, it will be created
automatically.

```yaml
apiVersion: sources.tanzu.vmware.com/v1alpha1
kind: HorizonSource
metadata:
  name: horizon-example
spec:
  sink:
    ref:
      apiVersion: eventing.knative.dev/v1
      kind: Broker
      name: example-broker
      namespace: default
  address: https://horizon.server.example.com
  skipTLSVerify: false
  secretRef:
    name: horizon-credentials
  serviceAccountName: horizon-source-sa
```

### Configure authentication

Create a Kubernetes `Secret` as per the name under `secretRef` in the
`HorizonSource` above which holds the required Horizon credentials. `domain`,
`username` and `password` are required fields. Replace the field values
accordingly.

```shell
kubectl create secret generic horizon-credentials --from-literal=domain="example.com" --from-literal=username="horizon-source-account" --from-literal=password='ReplaceMe'
```

### Deploy the Source

Finally, deploy the `HorizonSource`.

You should see a new deployment with the name `horizon-example-adapter` coming
up in the specified namespace (here `default`).


```shell
kubectl create -f horizon-source.yml

# wait for the deployment to become ready
kubectl wait --timeout=3m --for=condition=Available deploy/horizon-example-adapter
deployment.apps/horizon-example-adapter condition met
```

### Enable verbose (debug) logging

By default, each `HorizonSource` uses the `info` level for logging.

```shell
kubectl logs deploy/horizon-example-adapter
{"level":"warn","ts":"2022-07-05T09:59:02.701Z","logger":"horizon-source-adapter","caller":"v2/config.go:185","msg":"Tracing configuration is invalid, using the no-op default{error 26 0  empty json tracing config}","commit":"01ea50f"}
{"level":"warn","ts":"2022-07-05T09:59:02.701Z","logger":"horizon-source-adapter","caller":"v2/config.go:178","msg":"Sink timeout configuration is invalid, default to -1 (no timeout)","commit":"01ea50f"}
{"level":"warn","ts":"2022-07-05T09:59:02.701Z","logger":"horizon-source-adapter","caller":"horizon/horizon.go:130","msg":"using potentially insecure connection to Horizon API server","commit":"01ea50f","address":"https://horizon.server.example.com","insecure":true}
{"level":"info","ts":"2022-07-05T09:59:04.140Z","logger":"horizon-source-adapter","caller":"horizon/adapter.go:97","msg":"starting horizon source adapter","commit":"01ea50f","source":"https://horizon.server.example.com","pollIntervalSeconds":1}
```

To increase verbosity, update the logging configuration for the VMware Sources
and then perform a rolling restart of the `HorizonSource` adapter for the
logging changes to take effect.

```shell
# update general logging configuration
kubectl -n vmware-sources edit cm config-logging
```

A new window opens with an interactive editor.

Change the JSON line `"level": "info"` to `"level": "debug"`. Save and exit the
editor.

Perform a rolling restart of the running `HorizonSource`.

```shell
kubectl rollout restart deployment/horizon-example-adapter
```

