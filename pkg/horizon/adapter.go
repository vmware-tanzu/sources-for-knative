/*
Copyright 2022 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package horizon

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/jpillora/backoff"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"knative.dev/eventing/pkg/adapter/v2"
	"knative.dev/pkg/logging"
)

const (
	defaultPollInterval = time.Second
	eventTypeFormat     = "com.vmware.horizon.%s.v0"

	retryBackoff  = time.Second
	retryMaxTries = 5
)

type envConfig struct {
	// Include the standard adapter.EnvConfig used by all adapters.
	adapter.EnvConfig

	// Horizon settings
	Address  string `envconfig:"HORIZON_URL" required:"true"`
	Insecure bool   `envconfig:"HORIZON_INSECURE" default:"false"`
	// overwrite useful for local development
	SecretPath string `envconfig:"HORIZON_SECRET_PATH" default:""`
}

func NewEnv() adapter.EnvConfigAccessor { return &envConfig{} }

// Adapter reads events from the VMware Horizon API
type Adapter struct {
	client cloudevents.Client

	source       string
	sink         string
	hclient      Client
	clock        clock.Clock
	pollInterval time.Duration
}

func NewAdapter(ctx context.Context, _ adapter.EnvConfigAccessor, ceClient cloudevents.Client) adapter.Adapter {
	logger := logging.FromContext(ctx)

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		logger.Fatalw("process environment variables", zap.Error(err))
	}

	hc, err := newHorizonClient(ctx)
	if err != nil {
		logger.Fatalw("create horizon client", zap.Error(err))
	}

	return &Adapter{
		client:       ceClient,
		source:       env.Address,
		sink:         env.GetSink(),
		hclient:      hc,
		clock:        clock.New(),
		pollInterval: defaultPollInterval,
	}
}

// Start runs the adapter. Returns if ctx is cancelled or on unrecoverable
// error, e.g. reading or sending events.
func (a *Adapter) Start(ctx context.Context) error {
	return a.run(ctx)
}

// run starts polling the Horizon event API until the specified context is
// cancelled or when an error is returned while retrieving Horizon events
func (a *Adapter) run(ctx context.Context) error {
	var (
		lastEvent *AuditEventSummary
		since     Timestamp
	)

	logger := logging.FromContext(ctx).With(
		zap.String("source", a.source),
		zap.Duration("pollIntervalSeconds", a.pollInterval),
	)
	logger.Infow("starting horizon source adapter")

	ticker := a.clock.Ticker(a.pollInterval)
	defer func() {
		ticker.Stop()

		if err := a.hclient.Logout(context.Background()); err != nil {
			logger.Warn("could not logout from Horizon API", zap.Error(err))
		}
	}()

	backoffCfg := backoff.Backoff{
		Factor: 2,
		Jitter: false,
		Min:    retryBackoff,
		Max:    retryMaxTries * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			logger.Infof("stopping event stream")
			return ctx.Err()

		case <-ticker.C:
			if lastEvent != nil {
				since = Timestamp(lastEvent.Time)
			}

			if since == 0 {
				logger.Debug("retrieving initial set of events")
			} else {
				logger.Debugw("retrieving events with time range filter",
					zap.Any("sinceUnixMilli", since),
					zap.String("sinceConverted", time.Unix(int64(since/1000), 0).String()),
				)
			}

			events, err := a.hclient.GetEvents(ctx, since)
			if err != nil {
				return fmt.Errorf("get events: %w", err)
			}

			skip := false
			switch len(events) {
			case 0:
				skip = true
			case 1: // check if this is lastEvent we have already seen
				if lastEvent != nil && events[0].ID == lastEvent.ID {
					skip = true
				}
			}

			if skip {
				sleep := backoffCfg.Duration()
				logger.Debugw("backing off retrieving events: no new events received", zap.Duration("backoffSeconds", sleep))
				time.Sleep(sleep)
				continue
			}

			logger.Debugw("retrieved new events", zap.Int("count", len(events)))
			events = removeEvent(events, lastEvent)
			logger.Debugw("remaining new events after filtering out duplicate events", zap.Int("count", len(events)))
			lastEvent = a.sendEvents(ctx, events)
			backoffCfg.Reset()
		}
	}
}

// sendEvents sends the given events to the configured SINK returning the last
// successfully sent event
//
// TODO (@mgasch): There is a risk of poison pill issue here, leading to a constant loop
// in the invoking function.
func (a *Adapter) sendEvents(ctx context.Context, events []AuditEventSummary) *AuditEventSummary {
	logger := logging.FromContext(ctx).With(
		zap.String("source", a.source),
		zap.String("sink", a.sink),
	)

	// Horizon events are returned in descending time order thus the "id" in a
	// Horizon event can not be used for ordering (see testdata for example with
	// concurrent timestamps)
	reverse(events)

	// last successful processed event to track time offset in stream
	var lastEvent *AuditEventSummary

	ctx = cloudevents.ContextWithRetriesExponentialBackoff(ctx, retryBackoff, retryMaxTries)
	for i := range events {
		event := events[i]
		// don't waste cycles when ctx canceled
		if ctx.Err() != nil {
			return lastEvent
		}

		log := logger.With(zap.Any("event", event))
		ce, err := toCloudEvent(event, a.source)
		if err != nil {
			log.Errorw("skipping event because it could not be converted to cloudevent", zap.Error(err))
			continue
		}

		// TODO: better partial batch failure handling here?
		result := a.client.Send(ctx, ce)
		if !cloudevents.IsACK(result) {
			log.Errorw("could not send cloudevent", zap.Error(result))
			continue
		}
		log.Debugw("successfully sent event")
		lastEvent = &event
	}

	return lastEvent
}

// reverse mutates the given slice and reverses its order
func reverse(ev []AuditEventSummary) {
	for i := len(ev)/2 - 1; i >= 0; i-- {
		opp := len(ev) - 1 - i
		ev[i], ev[opp] = ev[opp], ev[i]
	}
}

// removeEvent returns a copy of list with the given event removed
func removeEvent(list []AuditEventSummary, event *AuditEventSummary) []AuditEventSummary {
	deduped := make([]AuditEventSummary, len(list))
	copy(deduped, list)

	if event == nil {
		return deduped
	}

	for i := range list {
		if list[i].ID == event.ID {
			// Remove the element at index i from a.
			copy(deduped[i:], deduped[i+1:])              // shift deduped[i+1:] left one index.
			deduped[len(deduped)-1] = AuditEventSummary{} // erase last element (write zero value).
			deduped = deduped[:len(deduped)-1]            // truncate slice.
		}
	}
	return deduped
}

func toCloudEvent(horizonEvent AuditEventSummary, source string) (cloudevents.Event, error) {
	ce := cloudevents.NewEvent()

	// TODO: revisit CE properties used here
	id := strconv.Itoa(int(horizonEvent.ID))
	ce.SetID(id)
	ce.SetSource(source)
	ce.SetType(convertEventType(horizonEvent.Type))
	t := time.Unix(horizonEvent.Time/1000, 0) // time is converted from ms
	ce.SetTime(t)

	if err := ce.SetData(cloudevents.ApplicationJSON, horizonEvent); err != nil {
		return cloudevents.Event{}, fmt.Errorf("set cloudevent data: %w", err)
	}

	if err := ce.Validate(); err != nil {
		return cloudevents.Event{}, fmt.Errorf("validation for cloudevent failed: %w", err)
	}

	return ce, nil
}

// convertEventType converts a Horizon event type to a normalized cloud event
// type. For example, VLSI_USERLOGGEDIN is converted to
// com.vmware.horizon.vlsi_userloggedin.v0
func convertEventType(t string) string {
	t = strings.ToLower(t)
	return fmt.Sprintf(eventTypeFormat, t)
}
