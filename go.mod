module github.com/vmware-tanzu/sources-for-knative

go 1.15

require (
	github.com/braintree/manners v0.0.0-20160418043613-82a8879fc5fd // indirect
	github.com/cloudevents/sdk-go/v2 v2.2.0
	github.com/codegangsta/cli v0.0.0-00010101000000-000000000000 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/elazarl/go-bindata-assetfs v1.0.0 // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/google/go-cmp v0.5.5
	github.com/hashicorp/go-multierror v1.1.0 // indirect
	github.com/jpillora/backoff v1.0.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kr/pty v1.1.8 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/vmware/govmomi v0.24.1-0.20210127152625-854ba4efe87e
	github.com/yudai/gotty v1.0.1
	github.com/yudai/hcl v0.0.0-20151013225006-5fa2393b3552 // indirect
	github.com/yudai/umutex v0.0.0-20150817080136-18216d265c6b // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/crypto v0.0.0-20201002170205-7f63de1d35b0
	gotest.tools v2.2.0+incompatible
	k8s.io/api v0.19.7
	k8s.io/apimachinery v0.19.7
	k8s.io/client-go v0.19.7
	k8s.io/code-generator v0.19.7
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	knative.dev/eventing v0.22.2
	knative.dev/hack v0.0.0-20210325223819-b6ab329907d3
	knative.dev/pkg v0.0.0-20210902173607-983897f9e37f
)

replace (
	github.com/codegangsta/cli => github.com/urfave/cli v1.19.1
	github.com/kr/pretty => github.com/dougm/pretty v0.0.0-20171025230240-2ee9d7453c02
)
