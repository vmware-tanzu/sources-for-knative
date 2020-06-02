/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

import (
	"bytes"
	"fmt"
	"testing"

	"gotest.tools/assert"
)

const (
	fakeVersion     = "fake-version"
	fakeBuildDate   = "fake-build-date"
	fakeGitRevision = "fake-git-revision"
)

func TestVersionSetup(t *testing.T) {
	versionCommand := NewVersionCommand()

	assert.Equal(t, versionCommand.Use, "version")
	assert.Equal(t, versionCommand.Short, "Prints the plugin version")
	assert.Assert(t, versionCommand.RunE != nil)
}

func TestVersionOutput(t *testing.T) {
	Version = fakeVersion
	BuildDate = fakeBuildDate
	GitRevision = fakeGitRevision
	expectedOutput := fmt.Sprintf(`Version:      %s
Build Date:   %s
Git Revision: %s
`, fakeVersion, fakeBuildDate, fakeGitRevision)

	output, err := runVersionCmd()

	assert.NilError(t, err)
	assert.Equal(t, output, expectedOutput)
}

func runVersionCmd() (string, error) {
	versionCmd := NewVersionCommand()

	output := new(bytes.Buffer)
	versionCmd.SetOut(output)
	err := versionCmd.Execute()
	return output.String(), err
}
