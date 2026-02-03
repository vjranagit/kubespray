package inventory

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/vjranagit/kubespray/pkg/config"
)

// Validator validates Ansible inventory files
type Validator struct {
	config *config.Config
}

// NewValidator creates a new inventory validator
func NewValidator(cfg *config.Config) *Validator {
	return &Validator{config: cfg}
}

// Validate checks if an inventory file is valid
func (v *Validator) Validate(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open inventory: %w", err)
	}
	defer file.Close()

	var (
		hasAll          bool
		hasControlPlane bool
		hasNode         bool
		hasEtcd         bool
	)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Check for required sections
		switch line {
		case "[all]":
			hasAll = true
		case "[kube_control_plane]", "[kube-master]":
			hasControlPlane = true
		case "[kube_node]", "[kube-node]":
			hasNode = true
		case "[etcd]":
			hasEtcd = true
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading inventory: %w", err)
	}

	// Validate required sections exist
	if !hasAll {
		return fmt.Errorf("inventory missing [all] section")
	}
	if !hasControlPlane {
		return fmt.Errorf("inventory missing [kube_control_plane] section")
	}
	if !hasNode {
		return fmt.Errorf("inventory missing [kube_node] section")
	}
	if !hasEtcd {
		return fmt.Errorf("inventory missing [etcd] section")
	}

	fmt.Printf("Inventory validation successful: %s\n", path)
	return nil
}
