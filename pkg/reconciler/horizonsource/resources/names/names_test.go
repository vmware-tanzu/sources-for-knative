package names

import (
	"testing"
)

func TestNewAdapterName(t *testing.T) {
	type args struct {
		source string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "source name within 63 char limit",
			args: args{
				source: "horizon-01",
			},
			want: "horizon-01-adapter",
		},
		{
			name: "source name exceeds 63 char limit",
			args: args{
				source: "horizon-01-7c6e3f71-7c98-43dc-b783-72bcf0103970-way-tooooooooooooooo-long",
			},
			want: "horizon-01-7c6e3f71-7c9f88c636a5f0781b807c2771ff73668ac-adapter",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewAdapterName(tt.args.source); got != tt.want {
				t.Errorf("NewAdapterName() = %v, want %v", got, tt.want)
			}
		})
	}
}
