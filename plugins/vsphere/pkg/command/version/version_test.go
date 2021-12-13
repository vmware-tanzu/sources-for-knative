/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package version_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/vmware-tanzu/sources-for-knative/plugins/vsphere/pkg/command/version"

	"gotest.tools/v3/assert"
)

const (
	fakeVersion     = "fake-version"
	fakeBuildDate   = "fake-build-date"
	fakeGitRevision = "fake-git-revision"
)

func TestVersionSetup(t *testing.T) {
	versionCommand := version.NewVersionCommand()

	assert.Equal(t, versionCommand.Use, "version")
	assert.Equal(t, versionCommand.Short, "Prints the plugin version")
	assert.Assert(t, versionCommand.RunE != nil)
}

func TestVersionOutput(t *testing.T) {
	version.Version = fakeVersion
	version.BuildDate = fakeBuildDate
	version.GitRevision = fakeGitRevision
	expectedOutput := fmt.Sprintf(`Version:      %s
Build Date:   %s
Git Revision: %s
`, fakeVersion, fakeBuildDate, fakeGitRevision)

	output, err := runVersionCmd()

	assert.NilError(t, err)
	assert.Equal(t, output, expectedOutput)
}

func runVersionCmd() (string, error) {
	versionCmd := version.NewVersionCommand()

	output := new(bytes.Buffer)
	versionCmd.SetOut(output)
	err := versionCmd.Execute()
	return output.String(), err
}
