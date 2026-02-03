package inventory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	
	"github.com/vjranagit/kubespray/pkg/config"
)

func TestGenerateInventory(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "hosts.ini")
	
	cfg := config.NewConfig()
	gen := NewGenerator(cfg)
	
	inv := &Inventory{
		Masters: []string{"192.168.1.10", "192.168.1.11"},
		Nodes:   []string{"192.168.1.20", "192.168.1.21", "192.168.1.22"},
		Etcd:    []string{"192.168.1.10", "192.168.1.11", "192.168.1.12"},
	}
	
	err := gen.Generate(inv, outputFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	// Verify file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatal("Inventory file was not created")
	}
	
	// Read and validate content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read inventory file: %v", err)
	}
	
	contentStr := string(content)
	
	// Check for master nodes
	for _, master := range inv.Masters {
		if !strings.Contains(contentStr, master) {
			t.Errorf("Master node %s not found in inventory", master)
		}
	}
	
	// Check for worker nodes
	for _, node := range inv.Nodes {
		if !strings.Contains(contentStr, node) {
			t.Errorf("Worker node %s not found in inventory", node)
		}
	}
}

func TestGenerateInventoryValidation(t *testing.T) {
	cfg := config.NewConfig()
	gen := NewGenerator(cfg)
	
	tests := []struct {
		name      string
		inventory *Inventory
		shouldErr bool
	}{
		{
			name: "Valid inventory",
			inventory: &Inventory{
				Masters: []string{"192.168.1.10"},
				Nodes:   []string{"192.168.1.20"},
			},
			shouldErr: false,
		},
		{
			name: "Empty masters",
			inventory: &Inventory{
				Masters: []string{},
				Nodes:   []string{"192.168.1.20"},
			},
			shouldErr: true,
		},
		{
			name: "Empty nodes",
			inventory: &Inventory{
				Masters: []string{"192.168.1.10"},
				Nodes:   []string{},
			},
			shouldErr: true,
		},
		{
			name: "Invalid IP address",
			inventory: &Inventory{
				Masters: []string{"invalid-ip"},
				Nodes:   []string{"192.168.1.20"},
			},
			shouldErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := filepath.Join(t.TempDir(), "hosts.ini")
			err := gen.Generate(tt.inventory, tmpFile)
			
			if tt.shouldErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestInventoryHostnameGeneration(t *testing.T) {
	cfg := config.NewConfig()
	gen := NewGenerator(cfg)
	
	inv := &Inventory{
		Masters: []string{"10.0.1.1", "10.0.1.2", "10.0.1.3"},
		Nodes:   []string{"10.0.2.1", "10.0.2.2"},
	}
	
	tmpFile := filepath.Join(t.TempDir(), "hosts.ini")
	err := gen.Generate(inv, tmpFile)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	
	content, _ := os.ReadFile(tmpFile)
	contentStr := string(content)
	
	// Should have hostnames like master-0, master-1, node-0, node-1
	expectedPatterns := []string{"master-", "node-"}
	for _, pattern := range expectedPatterns {
		if !strings.Contains(contentStr, pattern) {
			t.Errorf("Expected hostname pattern '%s' not found", pattern)
		}
	}
}
