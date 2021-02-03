/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"context"
	"reflect"
	"time"

	"github.com/vmware/govmomi/event"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
)

func newHistoryCollector(ctx context.Context, client *vim25.Client, begin time.Time) (*event.HistoryCollector, error) {
	mgr := event.NewManager(client)
	root := client.ServiceContent.RootFolder

	filter := types.EventFilterSpec{
		// everything
		Entity: &types.EventFilterSpecByEntity{
			Entity:    root,
			Recursion: types.EventFilterSpecRecursionOptionAll,
		},
		Time: &types.EventFilterSpecByTime{
			BeginTime: types.NewTime(begin),
		},
	}

	return mgr.CreateCollectorForEvents(ctx, filter)
}

// eventDetails contains the type and Class of an event received from vCenter
// supported event classes: event, eventex, extendedevent. Class to type
// mapping:
// event: retrieved from event Class, e.g. VmPoweredOnEvent
// eventex: retrieved from EventTypeId
// extendedevent: retrieved from EventTypeId
type eventDetails struct {
	Class string
	Type  string
}

// getEventDetails retrieves the underlying vSphere event class and name for
// the given BaseEvent, e.g. VmPoweredOnEvent (event) or
// com.vmware.applmgmt.backup.job.failed.event (extendedevent)
func getEventDetails(event types.BaseEvent) eventDetails {
	var details eventDetails

	switch e := event.(type) {
	case *types.EventEx:
		details.Class = "eventex"
		details.Type = e.EventTypeId
	case *types.ExtendedEvent:
		details.Class = "extendedevent"
		details.Type = e.EventTypeId
	default:
		t := reflect.TypeOf(event).Elem().Name()
		details.Class = "event"
		details.Type = t
	}

	return details
}
