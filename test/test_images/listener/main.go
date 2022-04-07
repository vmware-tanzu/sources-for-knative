/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"encoding/xml"
	"log"

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
	ExpectedEventCount int    `envconfig:"EVENT_COUNT"`
}

type VMPoweredOffEvent struct {
	XMLName xml.Name `xml:"VmPoweredOffEvent"`
	Message string   `xml:"fullFormattedMessage"`
}

func main() {
	var env envConfig
	if err := envconfig.Process("", &env); err != nil {
		log.Fatalf("Unable to read environment config: %v", err)
	}

	numExpectedEvents := env.ExpectedEventCount

	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatal(err.Error())
	}

	ctx := signals.NewContext()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	log := logging.FromContext(ctx)

	events := make(chan cloudevents.Event, numExpectedEvents)
	// receive events, putting them into the channel only if they meet the type we are expecting
	go func() {
		if err = client.StartReceiver(ctx, func(event cloudevents.Event) {
			log.Infof("Received event: %s", event.String())
			if event.Type() == env.ExpectedEventType {
				events <- event
			}
		}); err != nil {
			log.Fatalf("receiving events: %v", err)
		}
	}()

	count := 0
	// Process events one by one, keeping count. Exit when count is reached, and cancel the start receiver
	for event := range events {
		eventData := &VMPoweredOffEvent{}
		if err := xml.Unmarshal(event.Data(), eventData); err != nil {
			log.Fatalf("Failed to unmarshal CloudEvent xml data: ", err)
		}
		log.Infof("Raw Message: %s", eventData.Message)

		// assert required CE extension attributes are always present
		if event.Extensions()[ceVSphereEventClass] == "" {
			log.Fatalf("CloudEvent extension %q not set", ceVSphereEventClass)
		}
		if event.Extensions()[ceVSphereAPIKey] == "" {
			log.Fatalf("CloudEvent extension %q not set", ceVSphereAPIKey)
		}

		count += 1
		if count == numExpectedEvents {
			log.Infow("cancelling context: Received expected number of events")
			cancel()
			break
		}

	}

	log.Infow("successfully received event(s)", "count", count)
}
