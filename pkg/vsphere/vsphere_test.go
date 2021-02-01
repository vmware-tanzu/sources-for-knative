/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"reflect"
	"testing"

	"github.com/vmware/govmomi/vim25/types"
)

func Test_getEventDetails(t *testing.T) {
	type args struct {
		event types.BaseEvent
	}
	tests := []struct {
		name string
		args args
		want eventDetails
	}{
		{
			name: "VmPoweredOnEvent",
			args: args{&types.VmPoweredOnEvent{}},
			want: eventDetails{
				Class: "event",
				Type:  "VmPoweredOnEvent",
			},
		},
		{
			name: "EventEx",
			args: args{&types.EventEx{
				EventTypeId: "snapshotcreated.com.backup.provider.foo",
			}},
			want: eventDetails{
				Class: "eventex",
				Type:  "snapshotcreated.com.backup.provider.foo",
			},
		},
		{
			name: "ExtendedEvent",
			args: args{&types.ExtendedEvent{
				EventTypeId: "tokeninvalid.com.auth.provider.foo",
			}},
			want: eventDetails{
				Class: "extendedevent",
				Type:  "tokeninvalid.com.auth.provider.foo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getEventDetails(tt.args.event); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getEventDetails() = %v, want %v", got, tt.want)
			}
		})
	}
}
