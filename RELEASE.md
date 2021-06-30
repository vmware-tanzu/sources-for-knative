# Creating a release for VMware Tanzu Sources for Knative

## Verification

Before creating a release, verify its releasability with the corresponding
Github Action
[workflow](https://github.com/vmware-tanzu/sources-for-knative/actions/workflows/knative-releasability.yaml).

Enter the next `Release? (vX.Y)` value and (optionally) a value for `Slack
Channel? (release-#)` if you want to send the output to Slack.

Only if the release status is `GREEN`, proceed with the next steps.

## Create a `branch`

Through the Github UI, create a new release branch, e.g. `release-0.24`. In the
branch selection window, enter a release value following the example above and
click `"create branch: <release-name>"`.

## Creating a `tag`

We use `tags` to drive the creation of the releases. This is handled by release
workflow in [release.yaml](../.github/workflows/release.yaml).

`Tags` need to be pushed directly to upstream, so typical PR workflow will not
work. Here's one example workflow (assuming of course you have rights to
directly push to the upstream repo).

Create a temporary clone of the repo, e.g. to `/tmp`. Then create the `tag` and
push to origin. For example, to create a release called `v0.24.0`, you would run
the following commands:

```shell
cd /tmp
git clone git@github.com:vmware-tanzu/sources-for-knative.git
cd sources-for-knative

# checkout the branch created above
git checkout release-0.24

# associate the tag with the branch
git tag -a v0.24.0 -m "Release v0.24.0"
git push origin v0.24.0
```

## Release Notes

Release notes can be generated via the corresponding Github Actions
[workflow](.github/workflows/knative-release-notes.yaml). The workflow can be
triggered via manual invocation (`workflow_dispatch`)
[here](https://github.com/vmware-tanzu/sources-for-knative/actions/workflows/knative-release-notes.yaml).
