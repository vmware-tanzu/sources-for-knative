module github.com/vmware-tanzu/sources-for-knative

go 1.13

require (
	contrib.go.opencensus.io/exporter/ocagent v0.6.0 // indirect
	contrib.go.opencensus.io/exporter/stackdriver v0.12.9-0.20191108183826-59d068f8d8ff // indirect
	contrib.go.opencensus.io/exporter/zipkin v0.1.1 // indirect
	github.com/aws/aws-sdk-go v1.30.3 // indirect
	github.com/cloudevents/sdk-go/v2 v2.0.0-preview8
	github.com/gobuffalo/envy v1.9.0 // indirect
	github.com/golang/groupcache v0.0.0-20200121045136-8c9f03a8e57e // indirect
	github.com/golang/protobuf v1.3.5 // indirect
	github.com/google/go-cmp v0.4.0
	github.com/googleapis/gnostic v0.4.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway v1.14.3 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/imdario/mergo v0.3.9 // indirect
	github.com/json-iterator/go v1.1.9 // indirect
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/kr/text v0.2.0 // indirect
	github.com/markbates/inflect v1.0.4 // indirect
	github.com/openzipkin/zipkin-go v0.2.2 // indirect
	github.com/prometheus/client_golang v1.1.0 // indirect
	github.com/prometheus/common v0.9.1 // indirect
	github.com/prometheus/procfs v0.0.11 // indirect
	github.com/robfig/cron v1.2.0 // indirect
	github.com/rogpeppe/go-internal v1.5.2 // indirect
	github.com/vmware/govmomi v0.22.2
	go.opencensus.io v0.22.3 // indirect
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59 // indirect
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e // indirect
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a // indirect
	golang.org/x/sys v0.0.0-20200331124033-c3d80250170d // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0 // indirect
	golang.org/x/tools v0.0.0-20200402223321-bcf690261a44 // indirect
	gomodules.xyz/jsonpatch/v2 v2.1.0 // indirect
	google.golang.org/api v0.20.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/grpc v1.28.0 // indirect
	k8s.io/api v0.18.0
	k8s.io/apiextensions-apiserver v0.18.0 // indirect
	k8s.io/apimachinery v0.18.0
	k8s.io/client-go v0.18.0
	k8s.io/code-generator v0.18.0
	k8s.io/kube-openapi v0.0.0-20200121204235-bf4fb3bd569c
	k8s.io/utils v0.0.0-20200327001022-6496210b90e8 // indirect
	knative.dev/eventing v0.13.1-0.20200402224818-448b4e91c132
	knative.dev/pkg v0.0.0-20200427190051-6b9ee63b4aad
	knative.dev/test-infra v0.0.0-20200427194751-acea40b6aead // indirect
)

replace (
	github.com/kr/pretty => github.com/dougm/pretty v0.0.0-20171025230240-2ee9d7453c02
	k8s.io/api => k8s.io/api v0.16.4
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.16.4
	k8s.io/apimachinery => k8s.io/apimachinery v0.16.4
	k8s.io/client-go => k8s.io/client-go v0.16.4
	k8s.io/code-generator => k8s.io/code-generator v0.16.4
)
