/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/xml"
	"log"
	"strconv"
	"sync/atomic"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/kelseyhightower/envconfig"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

const (
	ceVSphereAPIKey     = "vsphereapiversion"
	ceVSphereEventClass = "eventclass"
)

type envConfig struct {
	ExpectedEventType  string `envconfig:"EVENT_TYPE"`
	ExpectedEventCount string `envconfig:"EVENT_COUNT"`
}

type VMPoweredOffEvent struct {
	XMLName xml.Name `xml:"VmPoweredOffEvent"`
	Message string   `xml:"fullFormattedMessage"`
}

func main() {
	ctx := signals.NewContext()

	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Unable to read environment config: %v", err)
	}

	numExpectedEvents, err := strconv.Atoi(env.ExpectedEventCount)
	if err != nil {
		log.Fatalf("Invalid value for EVENT_COUNT (%v): %v", env.ExpectedEventCount, err)
	}

	var count int32
	if err = client.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) {
		logging.FromContext(ctx).Infof("Received event: %s", event.String())

		if event.Type() == env.ExpectedEventType {
			atomic.AddInt32(&count, 1)

			eventData := &VMPoweredOffEvent{}
			if err := xml.Unmarshal(event.Data(), eventData); err != nil {
				logging.FromContext(ctx).Fatalf("Failed to unmarshal CloudEvent xml data: ", err)
			}
			logging.FromContext(ctx).Infof("Raw Message: %s", eventData.Message)
		}

		// assert required CE extension attributes are always present
		if event.Extensions()[ceVSphereEventClass] == "" {
			logging.FromContext(ctx).Fatalf("CloudEvent extension %q not set", ceVSphereEventClass)
		}
		if event.Extensions()[ceVSphereAPIKey] == "" {
			logging.FromContext(ctx).Fatalf("CloudEvent extension %q not set", ceVSphereAPIKey)
		}

		if count == int32(numExpectedEvents) {
			logging.FromContext(ctx).Infow("cancelling context: Received expected number of events")
			cancel()
		}

	}); err != nil {
		logging.FromContext(ctx).Fatalf("receiving events: %v", err)
	}

	if count == 0 {
		logging.FromContext(ctx).Fatalf("no events received")
	}
	logging.FromContext(ctx).Infow("successfully received event(s)", "count", count)
}
