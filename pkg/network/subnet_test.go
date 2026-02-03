package network

import (
	"net"
	"testing"
)

func TestCalculateSubnet(t *testing.T) {
	tests := []struct {
		name        string
		cidr        string
		expectedNet string
		shouldErr   bool
	}{
		{
			name:        "Valid /16 CIDR",
			cidr:        "10.233.0.0/16",
			expectedNet: "10.233.0.0",
			shouldErr:   false,
		},
		{
			name:        "Valid /24 CIDR",
			cidr:        "192.168.1.0/24",
			expectedNet: "192.168.1.0",
			shouldErr:   false,
		},
		{
			name:      "Invalid CIDR",
			cidr:      "invalid-cidr",
			shouldErr: true,
		},
		{
			name:      "Empty CIDR",
			cidr:      "",
			shouldErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ipNet, err := net.ParseCIDR(tt.cidr)
			
			if tt.shouldErr {
				if err == nil {
					t.Error("Expected error but got nil")
				}
				return
			}
			
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			
			if ipNet.IP.String() != tt.expectedNet {
				t.Errorf("Expected network %s, got %s", tt.expectedNet, ipNet.IP.String())
			}
		})
	}
}

func TestSubnetSplit(t *testing.T) {
	tests := []struct {
		name         string
		parentCIDR   string
		newPrefix    int
		expectedNets int
	}{
		{
			name:         "Split /16 into /18",
			parentCIDR:   "10.233.0.0/16",
			newPrefix:    18,
			expectedNets: 4,
		},
		{
			name:         "Split /24 into /26",
			parentCIDR:   "192.168.1.0/24",
			newPrefix:    26,
			expectedNets: 4,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, parent, err := net.ParseCIDR(tt.parentCIDR)
			if err != nil {
				t.Fatalf("Failed to parse CIDR: %v", err)
			}
			
			parentPrefix, _ := parent.Mask.Size()
			if tt.newPrefix <= parentPrefix {
				t.Skip("Invalid test: new prefix must be larger than parent")
			}
			
			// Calculate expected number of subnets
			expectedCount := 1 << (tt.newPrefix - parentPrefix)
			if expectedCount != tt.expectedNets {
				t.Errorf("Expected %d subnets, calculated %d", tt.expectedNets, expectedCount)
			}
		})
	}
}

func TestIPIncrement(t *testing.T) {
	tests := []struct {
		name     string
		startIP  string
		count    int
		expected string
	}{
		{
			name:     "Increment by 1",
			startIP:  "192.168.1.10",
			count:    1,
			expected: "192.168.1.11",
		},
		{
			name:     "Increment by 10",
			startIP:  "10.0.0.1",
			count:    10,
			expected: "10.0.0.11",
		},
		{
			name:     "Increment across boundary",
			startIP:  "192.168.1.255",
			count:    1,
			expected: "192.168.2.0",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.startIP)
			if ip == nil {
				t.Fatalf("Invalid start IP: %s", tt.startIP)
			}
			
			// Convert to 4-byte representation
			ip = ip.To4()
			if ip == nil {
				t.Fatal("Not an IPv4 address")
			}
			
			// Increment IP
			for i := 0; i < tt.count; i++ {
				for j := len(ip) - 1; j >= 0; j-- {
					ip[j]++
					if ip[j] != 0 {
						break
					}
				}
			}
			
			result := ip.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestSubnetContains(t *testing.T) {
	_, subnet, _ := net.ParseCIDR("10.233.0.0/16")
	
	tests := []struct {
		name     string
		ip       string
		contains bool
	}{
		{"Inside subnet", "10.233.1.1", true},
		{"At subnet start", "10.233.0.0", true},
		{"At subnet end", "10.233.255.255", true},
		{"Outside subnet", "10.234.0.0", false},
		{"Different network", "192.168.1.1", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("Invalid IP: %s", tt.ip)
			}
			
			contains := subnet.Contains(ip)
			if contains != tt.contains {
				t.Errorf("Expected Contains=%v, got %v", tt.contains, contains)
			}
		})
	}
}
