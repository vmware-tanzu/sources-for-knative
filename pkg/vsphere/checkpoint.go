/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	// replay history from this time by default
	CheckpointDefaultAge = 5 * time.Minute
	// create checkpoint every frequency but only on changes
	CheckpointDefaultPeriod = 10 * time.Second
	// key name used in KV store for storing the latest checkpoint
	checkpointKey = "checkpoint"
)

var (
	ErrInvalidInterval = errors.New("invalid checkpoint time interval")
)

// checkpoint represents a vCenter checkpoint object
type checkpoint struct {
	VCenter string `json:"vCenter"`
	// last vCenter event key successfully processed
	LastEventKey int32 `json:"lastEventKey"`
	// last event type, e.g. VmPoweredOffEvent useful for debugging
	LastEventType string `json:"lastEventType"`
	// last vCenter event key timestamp (UTC) successfully processed - used as
	// starting point for vCenter stream
	LastEventKeyTimestamp time.Time `json:"lastEventKeyTimestamp"`
	// timestamp (UTC) when this checkpoint was created
	CreatedTimestamp time.Time `json:"createdTimestamp"`
}

// CheckpointConfig influences the checkpoint behavior. It configures the
// maximum age of the replay (look-back) window when starting the event stream
// and the period of saving the checkpoint
type CheckpointConfig struct {
	// max replay window
	MaxAge time.Duration `json:"maxAge"`
	// create checkpoints at given frequency
	Period time.Duration `json:"period"`
}

// MarshalJSON defines custom marshalling logic to support human-readable time
// input on the checkpoint configuration, e.g. "10m" or "1h".
func (c *CheckpointConfig) MarshalJSON() ([]byte, error) {
	var out struct {
		MaxAge string `json:"maxAge"`
		Period string `json:"period"`
	}

	if c.MaxAge < time.Duration(0) {
		return nil, ErrInvalidInterval
	}

	if c.Period < time.Duration(0) {
		return nil, ErrInvalidInterval
	}

	out.MaxAge = c.MaxAge.String()
	out.Period = c.Period.String()
	return json.Marshal(out)
}

// UnmarshalJSON defines custom marshalling logic to support human-readable time
// input on the checkpoint configuration, e.g. "10m" or "1h". Using numbers
// without time suffix as input will fail encoding/decoding.
func (c *CheckpointConfig) UnmarshalJSON(b []byte) error {
	var in struct {
		MaxAge string `json:"maxAge"`
		Period string `json:"period"`
	}

	var (
		v   time.Duration
		err error
	)

	if err = json.Unmarshal(b, &in); err != nil {
		return err
	}

	if in.MaxAge == "" {
		v = CheckpointDefaultAge
	} else {
		v, err = time.ParseDuration(in.MaxAge)
		if err != nil {
			return err
		}

		if v < time.Duration(0) {
			return ErrInvalidInterval
		}
	}
	c.MaxAge = v

	if in.Period == "" {
		v = CheckpointDefaultPeriod
	} else {
		v, err = time.ParseDuration(in.Period)
		if err != nil {
			return err
		}
		if v < time.Duration(0) {
			return ErrInvalidInterval
		}
	}
	c.Period = v

	return nil
}

// newCheckpointConfig returns a checkpointConfig for the given JSON-encoded
// string. If the config is empty defaults for the event history replay window
// and frequency of saving the checkpoint will be used.
func newCheckpointConfig(config string) (*CheckpointConfig, error) {
	var c CheckpointConfig
	if err := json.Unmarshal([]byte(config), &c); err != nil {
		return nil, err
	}

	if c.Period == 0 {
		c.Period = CheckpointDefaultPeriod
	}

	return &c, nil
}
