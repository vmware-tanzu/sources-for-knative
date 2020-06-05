# Creating a release for VMware Tanzu Sources for Knative

We use tags to drive the creation of the releases. This is handled by release
workflow in [release.yaml](../.github/workflows/release.yaml).

## Creating a tag

Tags need to be pushed directly to upstream, so typical PR workflow will not
work. Here's one example workflow (assuming of course you have rights to
directly push to the upstream repo).

Create a temporary clone of the repo, to your /tmp, then create the tag and push
to origin. For example, to create a release called v0.15.0, you would run the
following commands:

```shell
cd /tmp
git clone git@github.com:vmware-tanzu/sources-for-knative.git
cd sources-for-knative
git tag -a v0.15.0 -m "Release v0.15.0"
git push origin v0.15.0
```
