package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()
	
	if cfg == nil {
		t.Fatal("Expected config, got nil")
	}
	
	if cfg.LogLevel != "info" {
		t.Errorf("Expected log_level 'info', got '%s'", cfg.LogLevel)
	}
	
	if cfg.Kubernetes.Version != "v1.29.0" {
		t.Errorf("Expected k8s version 'v1.29.0', got '%s'", cfg.Kubernetes.Version)
	}
	
	if cfg.Kubernetes.NetworkPlugin != "calico" {
		t.Errorf("Expected network plugin 'calico', got '%s'", cfg.Kubernetes.NetworkPlugin)
	}
}

func TestConfigStructure(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Config
		validate func(*testing.T, *Config)
	}{
		{
			name: "AWS configuration",
			setup: func() *Config {
				cfg := NewConfig()
				cfg.Cloud.AWS.Region = "us-west-2"
				cfg.Cloud.AWS.InstanceType = "t3.large"
				return cfg
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Cloud.AWS.Region != "us-west-2" {
					t.Errorf("Expected region 'us-west-2', got '%s'", cfg.Cloud.AWS.Region)
				}
				if cfg.Cloud.AWS.InstanceType != "t3.large" {
					t.Errorf("Expected instance type 't3.large', got '%s'", cfg.Cloud.AWS.InstanceType)
				}
			},
		},
		{
			name: "GCP configuration",
			setup: func() *Config {
				cfg := NewConfig()
				cfg.Cloud.GCP.Project = "my-project"
				cfg.Cloud.GCP.Zone = "us-central1-a"
				return cfg
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.Cloud.GCP.Project != "my-project" {
					t.Errorf("Expected project 'my-project', got '%s'", cfg.Cloud.GCP.Project)
				}
			},
		},
		{
			name: "SSH configuration",
			setup: func() *Config {
				cfg := NewConfig()
				cfg.SSH.User = "ubuntu"
				cfg.SSH.Port = 22
				cfg.SSH.KeyPath = "/tmp/test_key"
				return cfg
			},
			validate: func(t *testing.T, cfg *Config) {
				if cfg.SSH.User != "ubuntu" {
					t.Errorf("Expected user 'ubuntu', got '%s'", cfg.SSH.User)
				}
				if cfg.SSH.Port != 22 {
					t.Errorf("Expected port 22, got %d", cfg.SSH.Port)
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := tt.setup()
			tt.validate(t, cfg)
		})
	}
}

func TestKubesprayPathExpansion(t *testing.T) {
	cfg := NewConfig()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("Cannot get home directory: %v", err)
	}
	
	expected := filepath.Join(home, ".kubespray")
	if cfg.KubesprayPath != expected {
		t.Errorf("Expected kubespray_path '%s', got '%s'", expected, cfg.KubesprayPath)
	}
}

func TestOfflineConfig(t *testing.T) {
	cfg := NewConfig()
	
	if cfg.Offline.Workers != 20 {
		t.Errorf("Expected 20 workers, got %d", cfg.Offline.Workers)
	}
	
	if cfg.Offline.RetryCount != 3 {
		t.Errorf("Expected retry count 3, got %d", cfg.Offline.RetryCount)
	}
	
	if cfg.Offline.Registry.Port != 5000 {
		t.Errorf("Expected registry port 5000, got %d", cfg.Offline.Registry.Port)
	}
	
	if !cfg.Offline.Cache.Enabled {
		t.Error("Expected cache to be enabled by default")
	}
}
