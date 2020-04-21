## Using `VSphereBinding` to create a `PowerCLI` "Cloud Shell"

This sample builds on our [previous sample](../powercli/README.md) to show how
to use `VSphereBinding` to create a `PowerCLI` "Cloud Shell" by running the
`vmware/powerclicore` container image as a Knative Service.

### Pre-requisites

This sample assumes that you have a vSphere environment set up already with
credentials in a Secret named `vsphere-credentials`. For the remainder of the
sample we will assume you are within the environment setup for the
[`vcsim` sample](../vcsim/README.md).

### Create the Binding

We are going to use the following binding to authenticate our `PowerCLI` "Cloud
Shell":

```yaml
apiVersion: sources.tanzu.vmware.com/v1alpha1
kind: VSphereBinding
metadata:
  name: cloud-shell-binding
spec:
  # Apply to every Service labeled "role: cloud-power-shell" in
  # this namespace.
  subject:
    apiVersion: serving.knative.dev/v1
    kind: Service
    selector:
      matchLabels:
        role: cloud-power-shell

  # The address and credentials for vSphere.
  # If you aren't using the simulator, change this!
  address: https://vcsim.default.svc.cluster.local
  skipTLSVerify: true
  secretRef:
    name: vsphere-credentials
```

Once you have your binding ready, apply it with:

```shell
kubectl apply -f binding.yaml
```

### Building our "Cloud Shell" service.

For the "shell" part of our demo, we are going to make use of
[yudai/gotty](https://github.com/yudai/gotty). We are going to use the following
`ko` configuration (in `.ko.yaml`) to base `gotty` on `vmware/powerclicore`:

```yaml
...
baseImageOverrides:
  ...
  github.com/vmware-tanzu/sources-for-knative/vendor/github.com/yudai/gotty: docker.io/vmware/powerclicore

```

Then we are going to deploy `gotty` as a Knative Service as follows:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: cloud-power-shell
  labels:
    role: cloud-power-shell
spec:
  template:
    spec:
      containers:
        # The binary is gotty (based on vmware/powerclicore)
        - image: ko://github.com/vmware-tanzu/sources-for-knative/vendor/github.com/yudai/gotty
          args:
            # Tell gotty to enable interacting with the session.
            - -w
            # Launch Powershell and run our setup commands without exiting.
            - pwsh
            - -NoExit
            - -Command
            - |
              Set-PowerCLIConfiguration -InvalidCertificateAction Ignore -Confirm:$false | Out-Null
              Connect-VIServer -Server ([System.Uri]$env:GOVC_URL).Host -User $env:GOVC_USERNAME -Password $env:GOVC_PASSWORD
```

This Service authenticates `PowerCLI` using our injected credentials, and then
creates a session over a websocket that allows the user to interact with
`PowerCLI` over a websocket. You can deploy this service via:

```shell
ko apply -f service.yaml
```

Watch for this service to become ready via:

```shell
kubectl get ksvc cloud-power-shell
```

When it reports as `Ready`, open the URL in your browser and try running:

```shell
Get-VIEevent | Write-Host
```

You should see very similar results to our previous sample!

### Cleanup

```shell
kubectl delete -f service.yaml
kubectl delete -f binding.yaml
```
