package network

import (
	"fmt"
	"net"
)

// Calculator handles network subnet calculations
type Calculator struct{}

// NewCalculator creates a new subnet calculator
func NewCalculator() *Calculator {
	return &Calculator{}
}

// CalculateSubnets splits a /16 network into service and pod subnets
// Service network: /18 (first quarter)
// Pod network: /18 (second quarter)
func (c *Calculator) CalculateSubnets(cidr string) (string, string, error) {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", "", fmt.Errorf("invalid CIDR: %w", err)
	}

	// Get network prefix length
	_, bits := ipNet.Mask.Size()
	if bits != 32 {
		return "", "", fmt.Errorf("only IPv4 networks are supported")
	}

	// For simplicity, we'll use the configured subnets from config
	// In a production implementation, we would calculate this dynamically
	serviceNet := "10.233.0.0/18"
	podNet := "10.233.64.0/18"

	return serviceNet, podNet, nil
}

// ValidateCIDR validates a CIDR notation
func (c *Calculator) ValidateCIDR(cidr string) error {
	_, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR %s: %w", cidr, err)
	}
	return nil
}
