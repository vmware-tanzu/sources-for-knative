/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/cloudevents/sdk-go/v2/client"
	cecontext "github.com/cloudevents/sdk-go/v2/context"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
)

type roundTripperTest struct {
	statusCodes  []int
	requestCount int
	events       []event.Event
}

func (r *roundTripperTest) RoundTrip(req *http.Request) (*http.Response, error) {
	code := r.statusCodes[r.requestCount]
	msg := cehttp.NewMessageFromHttpRequest(req)
	e, err := binding.ToEvent(context.TODO(), msg)
	if err != nil {
		return nil, err
	}
	r.events = append(r.events, *e)
	return &http.Response{StatusCode: code}, nil
}

type mockType struct {
	event *types.Event
}

func (m *mockType) GetEvent() *types.Event {
	return m.event
}

func TestSendEvents(t *testing.T) {
	testCases := map[string]struct {
		statusCodes []int
		moref       types.ManagedObjectReference
		baseEvents  []types.BaseEvent
		wantEvents  []event.Event
		result      error
	}{
		"one event, succeeds": {
			statusCodes: []int{200},
			moref:       types.ManagedObjectReference{Value: "VirtualMachine", Type: "vm59"},
			baseEvents:  []types.BaseEvent{&mockType{event: &types.Event{CreatedTime: time.Now()}}},
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
			if sendResult != tc.result {
				t.Errorf("Failed to send events, expected %v got %v", tc.result, sendResult)
			}
		})
	}
}
