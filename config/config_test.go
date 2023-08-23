package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadComponents(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    Components
		wantErr bool
	}{
		{
			name: "TestLoadComponents",
			args: args{
				path: "../configuration",
			},
			want: Components{
				Gateway:            Component{Enabled: true},
				HostScanner:        Component{Enabled: true},
				Kollector:          Component{Enabled: true},
				Kubescape:          Component{Enabled: true},
				KubescapeScheduler: Component{Enabled: true},
				Kubevuln:           Component{Enabled: true},
				KubevulnScheduler:  Component{Enabled: true},
				Operator:           Component{Enabled: true},
				OtelCollector:      Component{Enabled: true},
				Storage:            Component{Enabled: true},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadComponents(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadComponents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLoadConfig(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    Config
		wantErr bool
	}{
		{
			name: "TestLoadConfig",
			args: args{
				path: "../configuration",
			},
			want: Config{
				Namespace:                "kubescape",
				RestAPIPort:              "4002",
				CleanUpRoutineInterval:   10 * time.Minute,
				ConcurrencyWorkers:       3,
				TriggerSecurityFramework: false,
				MatchingRulesFilename:    "/etc/config/matchingRules.json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadConfig(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
