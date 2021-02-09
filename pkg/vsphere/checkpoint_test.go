/*
Copyright 2020 VMware, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package vsphere

import (
	"reflect"
	"testing"
	"time"
)

func Test_checkpointConfig_UnmarshalJSON(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *CheckpointConfig
		wantErr bool
	}{
		{
			name: "valid config human-readable time",
			args: args{b: []byte(`{"maxAge":"1h","period":"10s"}`)},
			want: &CheckpointConfig{
				MaxAge: time.Hour,
				Period: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "invalid config with seconds-encoded time",
			args:    args{b: []byte(`{"maxAge":3600000000000,"period":10000000000}`)},
			want:    &CheckpointConfig{},
			wantErr: true,
		},
		{
			name:    "invalid config",
			args:    args{b: []byte(`{"maxAge":}`)},
			want:    &CheckpointConfig{},
			wantErr: true,
		},
		{
			name:    "invalid config (negative values)",
			args:    args{b: []byte(`{"maxAge":"-1ns","period":"-1m"}`)},
			want:    &CheckpointConfig{},
			wantErr: true,
		},
		{
			name: "empty config",
			args: args{b: []byte(`{}`)},
			want: &CheckpointConfig{
				MaxAge: CheckpointDefaultAge,
				Period: CheckpointDefaultPeriod,
			},
			wantErr: false,
		},
		{
			name: "empty config with zero values",
			args: args{b: []byte(`{"maxAge":"0s","period":"0s"}`)},
			want: &CheckpointConfig{
				MaxAge: time.Second * 0,
				Period: time.Second * 0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := &CheckpointConfig{}
			if err := got.UnmarshalJSON(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_checkpointConfig_MarshalJSON(t *testing.T) {
	type fields struct {
		MaxAge time.Duration
		Period time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name:    "empty config with zero values",
			fields:  fields{},
			want:    []byte(`{"maxAge":"0s","period":"0s"}`),
			wantErr: false,
		},
		{
			name: "default config values",
			fields: fields{
				MaxAge: CheckpointDefaultAge,
				Period: CheckpointDefaultPeriod,
			},
			want:    []byte(`{"maxAge":"5m0s","period":"10s"}`),
			wantErr: false,
		},
		{
			name: "invalid values",
			fields: fields{
				MaxAge: -1,
				Period: -2,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &CheckpointConfig{
				MaxAge: tt.fields.MaxAge,
				Period: tt.fields.Period,
			}
			got, err := c.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %s, want %s", got, tt.want)
			}
		})
	}
}

func Test_newCheckpointConfig(t *testing.T) {
	type args struct {
		config string
	}
	tests := []struct {
		name    string
		args    args
		want    *CheckpointConfig
		wantErr bool
	}{
		{
			name: "valid config human-readable time",
			args: args{config: `{"maxAge":"1h","period":"10s"}`},
			want: &CheckpointConfig{
				MaxAge: 1 * time.Hour,
				Period: 10 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "invalid config with seconds-encoded time",
			args:    args{config: `{"maxAge":3600000000000,"period":10000000000}`},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "invalid config",
			args:    args{config: `{"maxAge":}`},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty config (use defaults)",
			args: args{config: `{}`},
			want: &CheckpointConfig{
				MaxAge: CheckpointDefaultAge,
				Period: CheckpointDefaultPeriod,
			},
			wantErr: false,
		},
		{
			name: "config with zero values",
			args: args{config: `{"maxAge":"0s","period":"0s"}`},
			want: &CheckpointConfig{
				MaxAge: time.Duration(0),
				Period: CheckpointDefaultPeriod,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newCheckpointConfig(tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("newCheckpointConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newCheckpointConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
