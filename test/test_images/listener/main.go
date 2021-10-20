/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

const (
	ceVSphereAPIKey     = "vsphereapiversion"
	ceVSphereEventClass = "eventclass"
)

func main() {
	ctx := signals.NewContext()

	client, err := cloudevents.NewClientHTTP()
	if err != nil {
		log.Fatal(err.Error())
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Launch a go routine to avoid blocking
	go func() {
		timeout := 10 * time.Second
		<-time.After(timeout)
		logging.FromContext(ctx).Infow("cancelling context: timeout reached", "timeout", timeout)
		cancel()
	}()

	var count int32
	if err = client.StartReceiver(ctx, func(ctx context.Context, event cloudevents.Event) {
		logging.FromContext(ctx).Infof("Received event: %s", event.String())
		atomic.AddInt32(&count, 1)

		// assert required CE extension attributes are always present
		if event.Extensions()[ceVSphereEventClass] == "" {
			logging.FromContext(ctx).Fatalf("CloudEvent extension %q not set", ceVSphereEventClass)
		}
		if event.Extensions()[ceVSphereAPIKey] == "" {
			logging.FromContext(ctx).Fatalf("CloudEvent extension %q not set", ceVSphereAPIKey)
		}
	}); err != nil {
		logging.FromContext(ctx).Fatalf("receiving events: %v", err)
	}

	if count == 0 {
		logging.FromContext(ctx).Fatalf("no events received")
	}
	logging.FromContext(ctx).Infow("successfully received event(s)", "count", count)
}
