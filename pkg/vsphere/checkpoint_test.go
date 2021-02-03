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

func Test_newCheckpointConfig(t *testing.T) {
	type args struct {
		config string
	}
	tests := []struct {
		name    string
		args    args
		want    *checkpointConfig
		wantErr bool
	}{
		{
			name: "valid config human-readable time",
			args: args{config: `{"maxAge":"1h","period":"10s"}`},
			want: &checkpointConfig{
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
			want: &checkpointConfig{
				MaxAge: checkpointDefaultAge,
				Period: checkpointDefaultPeriod,
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
