/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package horizon

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	ce "github.com/cloudevents/sdk-go/v2"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

const (
	testEvents = "./testdata/audit_events.golden"
)

func TestAdapter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	ctx = logging.WithLogger(ctx, zaptest.NewLogger(t).Sugar())

	receiver := newSink(t, ctx)
	tr, err := ce.NewHTTP(ce.WithTarget(receiver.URL()))
	require.NoError(t, err)

	ceClient, err := ce.NewClient(tr)
	require.NoError(t, err)

	f, err := os.Open(testEvents)
	require.NoErrorf(t, err, "open golden file: %s", testEvents)

	var events []AuditEventSummary
	dec := json.NewDecoder(f)
	err = dec.Decode(&events)
	require.NoError(t, err, "JSON decode test events")

	a := &Adapter{
		client:       ceClient,
		source:       "http://api.horizon.corp.local",
		sink:         receiver.URL(),
		hclient:      &horizonMockClient{events: events},
		clock:        clock.New(),
		pollInterval: time.Millisecond * 100,
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err = a.Start(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()

		var (
			expect = len(events)

			counter       int
			lastTimestamp time.Time
		)

		for counter != expect {
			e := <-receiver.receiveChan
			// 	verify timestamps are ascending
			if lastTimestamp.IsZero() {
				lastTimestamp = e.Time()
			}

			require.GreaterOrEqual(t, e.Time().UnixMilli(), lastTimestamp.UnixMilli())
			counter++
		}
	}()

	wg.Wait()
}

func TestAdapterMain(t *testing.T) {
	// Use the test executable to simulate the cmd/adapter process if
	// environment var t.Name() is set to "main"
	// (see https://talks.golang.org/2014/testing.slide#23)
	if os.Getenv(t.Name()) == "main" {
		adapter.Main("horizon-source", NewEnv, NewAdapter)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	receiver := newSink(t, ctx)

	// Run a simulated adapter main using the test executable.
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run="+t.Name())
	cmd.Env = append(os.Environ(),
		t.Name()+"=main",
		"K_SINK="+receiver.URL(),
		"INTERVAL="+"1ms",
		"NAMESPACE=namespace",
		"NAME=name",
		`K_METRICS_CONFIG={"domain":"x", "component":"x", "prometheusport":0, "configmap":{}}`,
		`K_LOGGING_CONFIG={}`,
	)
	err := cmd.Start()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err = cmd.Wait(); err != nil {
			t.Logf("wait returned with error: %v", err)
		}
	}()
}

func Test_removeItem(t *testing.T) {
	t.Run("event is nil", func(t *testing.T) {
		ev := createFakeEvents(10)
		got := removeEvent(ev, nil)
		require.Equal(t, ev, got)
	})

	t.Run("empty events", func(t *testing.T) {
		got := removeEvent([]AuditEventSummary{}, &AuditEventSummary{})
		require.Equal(t, []AuditEventSummary{}, got)
	})

	t.Run("one duplicate event", func(t *testing.T) {
		ev := createFakeEvents(3)
		got := removeEvent(ev, &AuditEventSummary{ID: 10})
		require.Equal(t, ev[1:], got)
	})
}

// createFakeEvents creates returns a []AuditEventSummary where the ID of each
// element is set to 10 + current counter
func createFakeEvents(count int) []AuditEventSummary {
	events := make([]AuditEventSummary, count)
	for i := 0; i < count; i++ {
		events[i] = AuditEventSummary{ID: int64(10 + i)}
	}

	return events
}

type sink struct {
	listener    net.Listener
	client      ce.Client
	proto       *ce.HTTPProtocol
	receiveChan chan ce.Event
}

func newSink(t *testing.T, ctx context.Context) *sink {
	s := &sink{receiveChan: make(chan ce.Event)}

	var err error
	s.listener, err = net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s.proto, err = ce.NewHTTP(ce.WithListener(s.listener))
	require.NoError(t, err)

	s.client, err = ce.NewClient(s.proto)
	require.NoError(t, err)

	go func() {
		err = s.client.StartReceiver(ctx, func(ctx context.Context, e ce.Event) ce.Result {
			select {
			case s.receiveChan <- e:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
		require.NoError(t, err)
		close(s.receiveChan)
	}()

	return s
}

func (s *sink) URL() string { return "http://" + s.listener.Addr().String() }

type horizonMockClient struct {
	invocations int
	events      []AuditEventSummary // 10 items in golden file
}

func (h *horizonMockClient) GetEvents(ctx context.Context, since Timestamp) ([]AuditEventSummary, error) {
	h.invocations++

	// Horizon API returns events ordered from newest to oldest
	// note: concurrent events (by time) are not ordered by id (see golden file for example)
	switch h.invocations {
	case 1:
		// first half
		return h.events[5:], nil
	case 2:
		return h.events[2:5], nil
	case 3:
		// two most recent events
		return h.events[0:2], nil
	default:
		// always return newest event, triggers backoff
		return h.events[0:1], nil
	}
}

func (h *horizonMockClient) Logout(ctx context.Context) error {
	return nil
}
