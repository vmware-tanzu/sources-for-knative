# Creating a release for VMware Tanzu Sources for Knative

## Verification

Before creating a release, verify its releasability with the corresponding
Github Action
[workflow](https://github.com/vmware-tanzu/sources-for-knative/actions/workflows/knative-releasability.yaml).

Enter the next `Release? (vX.Y)` value and (optionally) a value for `Slack
Channel? (release-#)` if you want to send the output to Slack.

Only if the release status is `GREEN`, proceed with the next steps.

## Create a `branch`

Through the Github UI, create a new release branch, e.g. `release-1.0`. In the
branch selection window, enter a release value following the example above and
click `"create branch: <release-name>"`.

## Creating a `tag`

We use `tags` to drive the creation of the releases. This is handled by release
workflow in [release.yaml](.github/workflows/release.yaml).

`Tags` need to be pushed directly to upstream, so typical PR workflow will not
work. Here's one example workflow (assuming of course you have rights to
directly push to the upstream repo).

Create a temporary clone of the repo, e.g. to `/tmp`. Then create the `tag` and
push to origin. Despite Knative 1.0, the release tags in this repo are still versioned in the `v0.x.y` format and correspond to the Knative tags `knative-v1.x.y`.
This starts from `v0.27.0` == `knative-v1.0.0` and each MINOR version addition in the `knative-v1.x` tag corresponds to the same addition on the `0.x` tag.

PATCH versions are ignored in the branch name. In case a release version MUST be patched, just redo the release in the same release branch pointing to the correct commit and with the correct tag, i.e:

| Branch       | Knative Tag  | Normal Tag       |
|--------------|--------------|------------------|
| release-v1.1 | `v0.27.0`    | `knative-v1.0.x` |
| release-v1.1 | `v0.28.0`    | `knative-v1.1.x` |
| release-v1.1 | `v0.28.1`    | `knative-v1.1.x` |
| release-v1.2 | `v0.29.0`    | `knative-v1.2.x` |
| release-v1.3 | `v0.30.0`    | `knative-v1.3.x` |
and so on...

As a practical example, to create a release called `v0.27.0` you would run the following commands:

```shell
cd /tmp
git clone git@github.com:vmware-tanzu/sources-for-knative.git
cd sources-for-knative

# checkout the branch created above
git checkout release-1.0

# associate the tag with the branch
git tag -a v0.27.0 -m "Release v0.27.0"

# since knative version 1.0 we need to create an aditional tag that matches the
# knative-v1.x.y tag format
git tag -a knative-v1.0.0 -m "Release knative-v1.0.0"

git push origin v0.27.0
git push origin knative-v1.0.0
```

**Notes:**
- To trigger the release workflow you cannot push more than 2 tags at a time.
- After this process finish, don't forget to go to the [Release Section](https://github.com/vmware-tanzu/sources-for-knative/releases), edit the latest release and deselect the `this is a pre-release` option.

## Release Notes

Release notes can be generated via the corresponding Github Actions
[Workflow](.github/workflows/knative-release-notes.yaml). The workflow can be
triggered via manual invocation (`workflow_dispatch`) [here](https://github.com/vmware-tanzu/sources-for-knative/actions/workflows/knative-release-notes.yaml).
