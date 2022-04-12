/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	ce "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

const (
	ceVSphereAPIKey     = "vsphereapiversion"
	ceVSphereEventClass = "eventclass"
)

type envConfig struct {
	ExpectedEventType  string `envconfig:"EVENT_TYPE" required:"true"`
	ExpectedEventCount int    `envconfig:"EVENT_COUNT" required:"true"`
}

type VMPoweredOffEvent struct {
	XMLName xml.Name `xml:"VmPoweredOffEvent"`
	Message string   `xml:"fullFormattedMessage"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		panic("unable to read environment config: " + err.Error())
	}

	ctx := signals.NewContext()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := logging.FromContext(ctx)

	numExpectedEvents := env.ExpectedEventCount
	client, err := ce.NewClientHTTP()
	if err != nil {
		logger.Fatalw("could not create cloudevents client", zap.Error(err))
	}

	eg, egCtx := errgroup.WithContext(ctx)
	events := make(chan ce.Event, numExpectedEvents)

	// cloudevents http receiver
	eg.Go(func() error {
		logger.Info("starting cloudevents listener")
		// receive events, putting them into the channel only if they meet the type we are expecting
		return client.StartReceiver(egCtx, func(event ce.Event) {
			logger.Infow("received cloud event on listener", zap.String("event", event.String()))
			if event.Type() == env.ExpectedEventType {
				select {
				case events <- event:
				default:
					logger.Warn("could not send on events channel")

					// artificial throttle to not spam logs in case of hot loop
					time.Sleep(time.Second)
				}
				return
			}
			logger.Warnw(
				"ignoring event: unexpected event type received",
				zap.String("received", event.Type()),
				zap.String("expected", env.ExpectedEventType),
			)
		})
	})

	// thread-safe counter
	eg.Go(func() error {
		count := 0
		// Process events one by one, keeping count. Exit when count is reached, and cancel the start receiver

		for {
			select {
			case <-egCtx.Done():
				return egCtx.Err()
			case event := <-events:
				logger.Infow("received event on events channel", zap.String("message", event.String()))

				// assert required CE extension attributes are always present
				class := event.Extensions()[ceVSphereEventClass]
				if class == nil || class == "" {
					return fmt.Errorf("cloudevent extension %q not set", ceVSphereEventClass)
				}

				apiKey := event.Extensions()[ceVSphereAPIKey]
				if apiKey == nil || apiKey == "" {
					return fmt.Errorf("cloudevent extension %q not set", ceVSphereAPIKey)
				}

				count++
				if count == numExpectedEvents {
					logger.Infow(
						"cancelling context: received expected number of events",
						zap.Int("expected", numExpectedEvents),
						zap.Int("received", count),
					)
					cancel()
					return nil
				}
			}
		}
	})

	if err := eg.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatalw("Could not successfully receive expected events", zap.Error(err))
	}
	logger.Info("shutdown complete")
}
