/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/client"
	cecontext "github.com/cloudevents/sdk-go/v2/context"
	"github.com/cloudevents/sdk-go/v2/event"
	cehttp "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/google/go-cmp/cmp"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
	"knative.dev/pkg/kvstore"
)

const (
	source    = "https://vcenter.local/sdk"
	failNever = -1
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
	r.requestCount++
	return &http.Response{StatusCode: code}, nil
}

type mockType struct {
	event *types.Event
}

func (m *mockType) GetEvent() *types.Event {
	return m.event
}

func TestSendEvents(t *testing.T) {
	now := time.Now().UTC()
	type sendResult struct {
		count int
		err   error
	}

	events := createTestEvents(3, source, now)

	testCases := map[string]struct {
		statusCodes []int
		baseEvents  []types.BaseEvent
		wantEvents  []*event.Event
		result      sendResult
	}{
		"one event, succeeds": {
			statusCodes: createStatusCodes(1, failNever),
			baseEvents:  events.vEvents[:1],
			wantEvents:  events.ceEvents[:1],
			result: sendResult{
				count: 1,
				err:   nil,
			},
		},
		"one event, fails": {
			statusCodes: createStatusCodes(1, 0),
			baseEvents:  events.vEvents[:1],
			wantEvents:  events.ceEvents[:1],
			result: sendResult{
				count: 0,
				err:   errors.New("500: "),
			},
		},
		"two events, last fails": {
			statusCodes: createStatusCodes(2, 1),
			baseEvents:  events.vEvents[:2],
			wantEvents:  events.ceEvents[:2],
			result: sendResult{
				count: 1,
				err:   errors.New("500: "),
			},
		},
		"three events, second fails": {
			statusCodes: createStatusCodes(3, 1),
			baseEvents:  events.vEvents[:3],
			wantEvents:  events.ceEvents[:2],
			result: sendResult{
				count: 1, // send will stop after the first event which errors
				err:   errors.New("500: "),
			},
		},
		"three events, all succeed": {
			statusCodes: createStatusCodes(3, failNever),
			baseEvents:  events.vEvents[:3],
			wantEvents:  events.ceEvents[:3],
			result: sendResult{
				count: 3,
				err:   nil,
			},
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

			adapter := vAdapter{
				Logger:          logger.Sugar(),
				CEClient:        c,
				Source:          source,
				PayloadEncoding: cloudevents.ApplicationXML,
				VAPIVersion:     "6.7.0",
			}
			count, result := adapter.sendEvents(ctx, tc.baseEvents)

			if count != tc.result.count {
				t.Errorf("Unexpected event count from sendEvents, expected %v got %v", tc.result.count, count)
			}

			if tc.result.err == nil && result != nil {
				t.Error("Unexpected result from sendEvents, wanted no error got ", result)
			} else if tc.result.err != nil && result == nil {
				t.Error("Unexpected result from sendEvents, did not get expected error ", tc.result.err)
			} else if tc.result.err != nil && result != nil {
				if result.Error() != tc.result.err.Error() {
					t.Errorf("Unexpected result from sendEvents, expected %v got %v", tc.result, result)
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

type testEvents struct {
	vEvents  []types.BaseEvent
	ceEvents []*event.Event
}

func createTestEvents(count int, source string, createTime time.Time) testEvents {
	const (
		keyBegin = 1000
	)

	te := testEvents{
		vEvents:  make([]types.BaseEvent, count),
		ceEvents: make([]*event.Event, count),
	}

	for i := 0; i < count; i++ {
		id := keyBegin + i
		be := createBaseEvent(id, createTime)
		ce := createCloudEvent(source, strconv.Itoa(id), be, createTime)
		te.vEvents[i] = be
		te.ceEvents[i] = ce
	}
	return te
}

func createBaseEvent(id int, created time.Time) types.BaseEvent {
	return &mockType{
		event: &types.Event{
			Key:         int32(id),
			CreatedTime: created,
		},
	}
}

func createCloudEvent(eventSource string, eventID string, baseEvent types.BaseEvent, eventTime time.Time) *event.Event {
	details := getEventDetails(baseEvent)

	ev := cloudevents.NewEvent(cloudevents.VersionV1)

	ev.SetType(fmt.Sprintf(eventTypeFormat, details.Type))
	ev.SetTime(eventTime)
	ev.SetID(eventID)
	ev.SetSource(eventSource)
	ev.SetExtension(ceVSphereEventClass, details.Class)
	ev.SetExtension(ceVSphereAPIKey, "6.7.0")
	if err := ev.SetData("application/xml", baseEvent); err != nil {
		panic("Failed to SetData")
	}
	return &ev
}

func Test_getBeginFromCheckpoint(t *testing.T) {
	now := time.Now().UTC()

	type args struct {
		vcTime time.Time
		cp     checkpoint
		maxAge time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Time
	}{
		{
			name: "empty checkpoint (use vcTime)",
			args: args{
				vcTime: now,
				cp:     checkpoint{},
				maxAge: CheckpointDefaultAge,
			},
			want: now,
		},
		{
			name: "checkpoint too old (use CheckpointDefaultAge)",
			args: args{
				vcTime: now,
				cp: checkpoint{
					LastEventKey:          1234,
					LastEventKeyTimestamp: now.Add(time.Hour * -1),
				},
				maxAge: CheckpointDefaultAge,
			},
			want: now.Add(CheckpointDefaultAge * -1),
		},
		{
			name: "valid checkpoint within custom CheckpointConfig maxAge",
			args: args{
				vcTime: now,
				cp: checkpoint{
					LastEventKey:          1234,
					LastEventKeyTimestamp: now.Add(time.Hour * -1),
				},
				maxAge: time.Hour * 2,
			},
			want: now.Add(time.Hour * -1),
		},
	}
	for _, tt := range tests {
		ctx := context.TODO()
		t.Run(tt.name, func(t *testing.T) {
			if got := getBeginFromCheckpoint(ctx, tt.args.vcTime, tt.args.cp, tt.args.maxAge); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getBeginFromCheckpoint() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_vAdapter_run(t *testing.T) {
	const (
		// number of vcsim events emitted for default VPX model
		vcsimEvents = 26
	)

	now := time.Now().UTC()

	type fields struct {
		StatusCodes []int
		Source      string
		KVStore     kvstore.Interface
		CpConfig    CheckpointConfig
	}
	tests := []struct {
		name              string
		fields            fields
		wantCheckpointKey int32 // key we expect in checkpoint after run returns
		wantRunErr        error // error we expect after run returns
	}{
		{
			name: "no existing checkpoint, no events received",
			fields: fields{
				StatusCodes: nil, // we don't send any events
				Source:      source,
				KVStore:     &fakeKVStore{},
				CpConfig: CheckpointConfig{
					MaxAge: CheckpointDefaultAge,
					Period: time.Millisecond,
				},
			},
			wantCheckpointKey: 0, // we never checkpoint in this test
			wantRunErr:        context.Canceled,
		},
		{
			name: "existing checkpoint, events received and all sends succeed",
			fields: fields{
				StatusCodes: createStatusCodes(vcsimEvents, failNever),
				Source:      source,
				KVStore: &fakeKVStore{
					data: map[string]string{
						checkpointKey: createCheckpoint(t, now.Add(time.Hour*-1)),
					},
					dataChan: make(chan string, 1),
				},
				CpConfig: CheckpointConfig{
					MaxAge: time.Hour,
					Period: time.Millisecond,
				},
			},
			wantCheckpointKey: 26,
			wantRunErr:        context.Canceled,
		},
		{
			name: "existing checkpoint, events received and first two sends succeeds",
			fields: fields{
				StatusCodes: createStatusCodes(vcsimEvents, 2),
				Source:      source,
				KVStore: &fakeKVStore{
					data: map[string]string{
						checkpointKey: createCheckpoint(t, now.Add(time.Hour*-1)),
					},
					dataChan: make(chan string, 1),
				},
				CpConfig: CheckpointConfig{
					MaxAge: time.Hour,
					Period: time.Millisecond,
				},
			},
			wantCheckpointKey: 2,
			wantRunErr:        context.Canceled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			simulator.Run(func(ctx context.Context, vim *vim25.Client) error {
				ctx = cecontext.WithTarget(ctx, "fake.example.com")

				roundTripper := &roundTripperTest{statusCodes: tt.fields.StatusCodes}
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

				vcClient := govmomi.Client{
					Client:         vim,
					SessionManager: session.NewManager(vim),
				}
				a := &vAdapter{
					Logger:   logger.Sugar(),
					Source:   tt.fields.Source,
					VClient:  &vcClient,
					CEClient: c,
					KVStore:  tt.fields.KVStore,
					CpConfig: tt.fields.CpConfig,
				}

				ctx, cancel := context.WithCancel(ctx)
				defer cancel()

				var (
					wg sync.WaitGroup
					// assertion variables
					cp     checkpoint
					runErr error
				)

				// run components
				wg.Add(1)
				go func() {
					defer wg.Done()
					runErr = a.run(ctx) // will be stopped with cancel()
				}()

				wg.Add(1)
				go func() {
					defer wg.Done()

					select {
					case data := <-tt.fields.KVStore.(*fakeKVStore).dataChan:
						err := json.Unmarshal([]byte(data), &cp)
						if err != nil {
							t.Errorf("unmarshal data from KV store: %v", err)
						}
						cancel() // stop run
					case <-ctx.Done():
					}
				}()

				// 	for test case(s) where we never send/checkpoint events so test won't hang
				if tt.wantCheckpointKey == 0 {
					wg.Add(1)
					go func() {
						defer wg.Done()
						time.Sleep(time.Millisecond * 100)
						cancel()
					}()
				}

				wg.Wait()

				if !reflect.DeepEqual(runErr, tt.wantRunErr) {
					// hack because govmomi does not wrap context.Canceled err (uses url.Error with
					// random port)
					if runErr != nil && !strings.Contains(runErr.Error(), "context canceled") {
						t.Error("run() unexpected error: ", runErr)
					}
				}

				if tt.wantCheckpointKey != cp.LastEventKey {
					t.Errorf("run() checkpointKey = %v, wantEventKey %v", cp.LastEventKey, tt.wantCheckpointKey)
				}

				return nil
			})
		})
	}
}

func createCheckpoint(t *testing.T, lastEventTS time.Time) string {
	t.Helper()
	cp := checkpoint{
		VCenter:               "",
		LastEventKey:          0,
		LastEventType:         "",
		LastEventKeyTimestamp: lastEventTS,
		CreatedTimestamp:      lastEventTS,
	}
	b, err := json.Marshal(cp)
	if err != nil {
		t.Fatalf("marshal checkpoint: %v", err)
	}

	return string(b)
}

// createStatusCodes returns a slice of status codes with count elements. If
// failAt is < 0, all status codes will be 200. If failAt is >= 0 failAt and
// following status codes of the returned slice will be 500s.
func createStatusCodes(count, failAt int) []int {
	code := 200
	codes := make([]int, count)

	for i := 0; i < count; i++ {
		if failAt == i {
			code = 500
		}
		codes[i] = code
	}

	return codes
}

type fakeKVStore struct {
	sync.Mutex
	data  map[string]string
	saved bool

	// send last checkpoint saved over this channel (should be buffered)
	// can be used so sync between read/write goroutines in tests
	dataChan chan string
}

func (f *fakeKVStore) Init(ctx context.Context) error {
	panic("implement me")
}

func (f *fakeKVStore) Load(ctx context.Context) error {
	panic("implement me")
}

func (f *fakeKVStore) Save(ctx context.Context) error {
	f.Lock()
	defer f.Unlock()
	if f.saved {
		return nil
	}
	f.saved = true
	f.dataChan <- f.data[checkpointKey]
	return nil
}

func (f *fakeKVStore) Get(ctx context.Context, key string, value interface{}) error {
	v, ok := f.data[key]
	if !ok {
		return fmt.Errorf("key %s does not exist", key)
	}
	err := json.Unmarshal([]byte(v), value)
	if err != nil {
		return fmt.Errorf("failed to Unmarshal %q: %w", v, err)
	}
	return nil
}

func (f *fakeKVStore) Set(ctx context.Context, key string, value interface{}) error {
	f.Lock()
	defer f.Unlock()

	f.saved = false
	bytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to Marshal: %w", err)
	}
	if f.data == nil {
		f.data = map[string]string{}
	}
	f.data[key] = string(bytes)
	return nil
}
