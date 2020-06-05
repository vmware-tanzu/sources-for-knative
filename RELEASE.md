# Creating a release for VMware Tanzu Sources for Knative

We use tags to drive the creation of the releases. This is handled by release
workflow in [release.yaml](../.github/workflows/release.yaml).

## Creating a tag

For example, to create a release called v0.15.0, you would run the following
command:

```shell
git tag -a v0.15.0 -m "Relase v0.15.0"
git push origin v0.15.0
```

