/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/client"
	cecontext "github.com/cloudevents/sdk-go/v2/context"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/google/go-cmp/cmp"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

type roundTripperTest struct {
	statusCodes  []int
	requestCount int
	events       []*event.Event
}

func (r *roundTripperTest) RoundTrip(req *http.Request) (*http.Response, error) {
	code := r.statusCodes[r.requestCount]
	msg := cehttp.NewMessageFromHttpRequest(req)
	e, err := binding.ToEvent(context.TODO(), msg)
	if err != nil {
		return nil, err
	}
	r.events = append(r.events, e)
	return &http.Response{StatusCode: code}, nil
}

type mockType struct {
	event *types.Event
}

func (m *mockType) GetEvent() *types.Event {
	return m.event
}

func TestSendEvents(t *testing.T) {
	now := time.Now()
	baseEvent := &mockType{event: &types.Event{CreatedTime: now}}
	testCases := map[string]struct {
		statusCodes []int
		moref       types.ManagedObjectReference
		baseEvents  []types.BaseEvent
		wantEvents  []*event.Event
		result      error
	}{
		"one event, succeeds": {
			statusCodes: []int{200},
			moref:       types.ManagedObjectReference{Value: "VirtualMachine", Type: "vm59"},
			baseEvents:  []types.BaseEvent{baseEvent},
			wantEvents:  []*event.Event{createEvent("test-source", "mockType", "0", "eventex", nil, baseEvent, now)},
		},
		"one event, fails": {
			statusCodes: []int{500},
			moref:       types.ManagedObjectReference{Value: "VirtualMachine", Type: "vm59"},
			baseEvents:  []types.BaseEvent{baseEvent},
			wantEvents:  []*event.Event{createEvent("test-source", "mockType", "0", "eventex", nil, baseEvent, now)},
			result:      errors.New("500: "),
		},
	}
	for n, tc := range testCases {
		ctx := context.Background()
		ctx = cecontext.WithTarget(ctx, "fake.example.com")
		t.Run(n, func(t *testing.T) {
			roundTripper := &roundTripperTest{statusCodes: tc.statusCodes}
			opts := []cehttp.Option{
				cehttp.WithRoundTripper(roundTripper),
			}
			p, err := cehttp.New(opts...)
			if err != nil {
				t.Error(err)
			}
			c, err := client.New(p, client.WithTimeNow(), client.WithUUIDs())
			if err != nil {
				t.Error(err)
			}
			logger := zaptest.NewLogger(t, zaptest.WrapOptions(zap.AddCaller()))

			adapter := vAdapter{Logger: logger.Sugar(), CEClient: c, Source: "test-source"}
			sendEvents := adapter.sendEvents(ctx)
			sendResult := sendEvents(tc.moref, tc.baseEvents)
			if tc.result == nil && sendResult != nil {
				t.Error("Unexpected result from sendEvents, wanted no error got ", sendResult)
			} else if tc.result != nil && sendResult == nil {
				t.Error("Unexpected result from sendEvents, did not get expected error ", tc.result)
			} else if tc.result != nil && sendResult != nil {
				if sendResult.Error() != tc.result.Error() {
					t.Errorf("Unexpected result from sendEvents, expected %v got %v", tc.result, sendResult)
				}
			}
			for i := range tc.wantEvents {
				if diff := cmp.Diff(tc.wantEvents[i], roundTripper.events[i]); diff != "" {
					t.Error("unexpected diff in events", diff)
				}
			}
		})
	}
}

func createEvent(eventSource, eventType, eventID, extensionKey string, extensionValue interface{}, baseEvent interface{}, eventTime time.Time) *event.Event {
	ev := cloudevents.NewEvent(cloudevents.VersionV1)

	ev.SetType("com.vmware.vsphere." + eventType)
	ev.SetTime(eventTime)
	ev.SetID(eventID)
	ev.SetSource(eventSource)
	if extensionKey != "" {
		ev.SetExtension(extensionKey, extensionValue)
	}
	if err := ev.SetData("application/xml", baseEvent); err != nil {
		panic("Failed to SetData")
	}
	return &ev
}
