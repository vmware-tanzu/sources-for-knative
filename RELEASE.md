# Creating a release for VMware Tanzu Sources for Knative

## Create a `branch`

Through the Github UI (or via git cli), create a new release branch, e.g. `release-1.10`. In the
branch selection window, enter a release value following the example above and
click `"create branch: <release-name>"`.

## Verification

Before creating a release, verify its releasability with the corresponding
Github Action
[workflow](https://github.com/vmware-tanzu/sources-for-knative/actions/workflows/knative-releasability.yaml).

Enter the next
- `Release? (vX.Y)` value (eg. "v1.12")
- `Module Release? (vX.Y)` value (eg. "v0.39")
- (optionally) a value for `Slack Channel? (release-#)` if you want to send the output to Slack.

Only if the release status is `GREEN`, proceed with the next steps.

## Creating a `tag`

We use `tags` to drive the creation of the releases. This is handled by release
workflow in [release.yaml](.github/workflows/release.yaml).

`Tags` need to be pushed directly to upstream, so typical PR workflow will not
work. Here's one example workflow (assuming of course you have rights to
directly push to the upstream repo).

Despite Knative's external semantic version uses a MAJOR version like 1.0, the release tags in this repo are still versioned in the `v0.x.y` format and correspond to the Knative tags `knative-v1.a.y`.
This starts from `v0.37.0` == `knative-v1.10.0` and each MINOR version addition of a `knative-v1.a.y` tag corresponds to the same addition of a `0.x.y` tag.

PATCH versions are ignored in the branch name. In case a release version MUST be patched, use the existing `release-MAJOR.MINOR` branch with a new tag pointing to the correct commit, i.e:

| Branch      | Normal Tag  | Knative Tag      |
|-------------|-------------|------------------|
| release-1.10 | `v0.37.0`   | `knative-v1.10.0` |
| release-1.11 | `v0.38.0`   | `knative-v1.11.0` |
| release-1.11 | `v0.38.1`   | `knative-v1.11.1` |
| release-1.12 | `v0.39.0`   | `knative-v1.12.0` |
| release-1.13 | `v0.40.0`   | `knative-v1.40.0` |
and so on...

As a practical example, to create a release for `v0.37.0` you would run the following commands:

```shell
cd /tmp
git clone git@github.com:vmware-tanzu/sources-for-knative.git
cd sources-for-knative

# checkout the branch created above
git checkout release-1.0

# associate the tag with the branch
git tag -a v0.37.0 -m "Release v0.37.0"

# since knative version 1.0 we need to create an aditional tag that matches the
# knative-v1.x.y tag format
git tag -a knative-v1.10.0 -m "Release knative-v1.10.0"

git push origin v0.37.0 knative-v1.10.0
```

## Release Notes

Release notes can be generated via the corresponding Github Actions
[workflow](.github/workflows/knative-release-notes.yaml). The workflow can be
triggered via manual invocation (`workflow_dispatch`) [here](https://github.com/vmware-tanzu/sources-for-knative/actions/workflows/knative-release-notes.yaml).

## Patch Release

To create a `PATCH` Release, go to the release target branch/branches and do a `git cherry-pick` of the desired commits:

```shel
# the patch release is done on the same release branch, no PATCH versions needed in the branch
git checkout release-1.0
git cherry-pick commit1 commit2 ...

# here the PATCH version is required and it must not exist already
git tag -a v0.37.1 -m "Release v0.37.1"

git tag -a knative-v1.10.1 -m "Release knative-v1.10.1"

git push origin v0.37.1 knative-v1.10.1
```

**Notes:**
- To trigger the release workflow you cannot push more than 2 tags at a time.

## Update and Upgrade deps

To update deps run:
```
./hack/update-deps.sh
```

To upgrade deps for a new release run:
```
./hack/update-deps.sh --upgrade --release v1.11 --module-release v0.38
```
**Notes:**
- Create a PR and fix the compatibility erros that may arrise
