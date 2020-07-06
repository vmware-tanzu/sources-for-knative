/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package command

func MutuallyExclusiveStringFlags(value1, value2 string, rest ...string) bool {
	nonEmpty := 0
	incrementIfNonEmpty(value1, &nonEmpty)
	incrementIfNonEmpty(value2, &nonEmpty)
	if nonEmpty > 1 {
		return false
	}
	for _, value := range rest {
		incrementIfNonEmpty(value, &nonEmpty)
		if nonEmpty > 1 {
			return false
		}
	}
	return true
}

func incrementIfNonEmpty(value string, nonEmpty *int) {
	if value != "" {
		*nonEmpty++
	}
}
