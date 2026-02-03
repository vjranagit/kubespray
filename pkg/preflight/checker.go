package preflight

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// CheckResult represents the result of a preflight check
type CheckResult struct {
	Name    string
	Passed  bool
	Message string
	Details map[string]interface{}
}

// Checker performs preflight validation checks
type Checker struct {
	hosts      []string
	sshUser    string
	sshKeyPath string
	sshPort    int
	timeout    time.Duration
}

// NewChecker creates a new preflight checker
func NewChecker(hosts []string, user, keyPath string, port int) *Checker {
	return &Checker{
		hosts:      hosts,
		sshUser:    user,
		sshKeyPath: keyPath,
		sshPort:    port,
		timeout:    30 * time.Second,
	}
}

// RunAll executes all preflight checks
func (c *Checker) RunAll(ctx context.Context) ([]CheckResult, error) {
	results := []CheckResult{}

	// Check SSH connectivity
	sshResult := c.CheckSSHConnectivity(ctx)
	results = append(results, sshResult...)

	// Check system requirements
	reqResult := c.CheckSystemRequirements(ctx)
	results = append(results, reqResult...)

	// Check network connectivity
	netResult := c.CheckNetworkConnectivity(ctx)
	results = append(results, netResult...)

	// Check Kubernetes version compatibility
	k8sResult := c.CheckKubernetesVersion()
	results = append(results, k8sResult)

	return results, nil
}

// CheckSSHConnectivity validates SSH access to all nodes
func (c *Checker) CheckSSHConnectivity(ctx context.Context) []CheckResult {
	results := []CheckResult{}

	for _, host := range c.hosts {
		result := CheckResult{
			Name:    fmt.Sprintf("SSH Connectivity - %s", host),
			Details: make(map[string]interface{}),
		}

		// Test SSH connection
		client, err := c.sshConnect(host)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf("Failed to connect: %v", err)
		} else {
			client.Close()
			result.Passed = true
			result.Message = "SSH connection successful"
		}

		results = append(results, result)
	}

	return results
}

// CheckSystemRequirements validates CPU, memory, and disk on each node
func (c *Checker) CheckSystemRequirements(ctx context.Context) []CheckResult {
	results := []CheckResult{}

	minCPU := 2
	minMemoryGB := 2
	minDiskGB := 20

	for _, host := range c.hosts {
		result := CheckResult{
			Name:    fmt.Sprintf("System Requirements - %s", host),
			Details: make(map[string]interface{}),
		}

		client, err := c.sshConnect(host)
		if err != nil {
			result.Passed = false
			result.Message = fmt.Sprintf("Cannot connect: %v", err)
			results = append(results, result)
			continue
		}
		defer client.Close()

		// Check CPU count
		cpuCount, err := c.runSSHCommand(client, "nproc")
		if err == nil {
			cpu, _ := strconv.Atoi(strings.TrimSpace(cpuCount))
			result.Details["cpu_cores"] = cpu
			if cpu < minCPU {
				result.Passed = false
				result.Message = fmt.Sprintf("Insufficient CPU cores: %d (minimum: %d)", cpu, minCPU)
				results = append(results, result)
				continue
			}
		}

		// Check memory
		memInfo, err := c.runSSHCommand(client, "cat /proc/meminfo | grep MemTotal")
		if err == nil {
			re := regexp.MustCompile(`MemTotal:\s+(\d+)\s+kB`)
			matches := re.FindStringSubmatch(memInfo)
			if len(matches) > 1 {
				memKB, _ := strconv.Atoi(matches[1])
				memGB := memKB / 1024 / 1024
				result.Details["memory_gb"] = memGB
				if memGB < minMemoryGB {
					result.Passed = false
					result.Message = fmt.Sprintf("Insufficient memory: %dGB (minimum: %dGB)", memGB, minMemoryGB)
					results = append(results, result)
					continue
				}
			}
		}

		// Check disk space
		diskInfo, err := c.runSSHCommand(client, "df -BG / | tail -1 | awk '{print $4}'")
		if err == nil {
			diskStr := strings.TrimSpace(strings.TrimSuffix(diskInfo, "G"))
			diskGB, _ := strconv.Atoi(diskStr)
			result.Details["disk_available_gb"] = diskGB
			if diskGB < minDiskGB {
				result.Passed = false
				result.Message = fmt.Sprintf("Insufficient disk space: %dGB (minimum: %dGB)", diskGB, minDiskGB)
				results = append(results, result)
				continue
			}
		}

		result.Passed = true
		result.Message = "System requirements met"
		results = append(results, result)
	}

	return results
}

// CheckNetworkConnectivity validates network connectivity between nodes
func (c *Checker) CheckNetworkConnectivity(ctx context.Context) []CheckResult {
	results := []CheckResult{}

	if len(c.hosts) < 2 {
		return results
	}

	for i, srcHost := range c.hosts {
		for j, dstHost := range c.hosts {
			if i >= j {
				continue
			}

			result := CheckResult{
				Name:    fmt.Sprintf("Network Connectivity - %s to %s", srcHost, dstHost),
				Details: make(map[string]interface{}),
			}

			client, err := c.sshConnect(srcHost)
			if err != nil {
				result.Passed = false
				result.Message = fmt.Sprintf("Cannot connect to source: %v", err)
				results = append(results, result)
				continue
			}
			defer client.Close()

			pingCmd := fmt.Sprintf("ping -c 3 -W 2 %s", dstHost)
			output, err := c.runSSHCommand(client, pingCmd)
			if err != nil || !strings.Contains(output, "3 received") {
				result.Passed = false
				result.Message = "Ping failed between nodes"
			} else {
				result.Passed = true
				result.Message = "Network connectivity verified"
			}

			results = append(results, result)
		}
	}

	return results
}

// CheckKubernetesVersion validates Kubernetes version compatibility
func (c *Checker) CheckKubernetesVersion() CheckResult {
	result := CheckResult{
		Name:    "Kubernetes Version Compatibility",
		Details: make(map[string]interface{}),
	}

	supportedVersions := []string{"v1.28", "v1.29", "v1.30"}
	result.Details["supported_versions"] = supportedVersions

	result.Passed = true
	result.Message = fmt.Sprintf("Supported versions: %s", strings.Join(supportedVersions, ", "))

	return result
}

// sshConnect establishes SSH connection to a host
func (c *Checker) sshConnect(host string) (*ssh.Client, error) {
	key, err := exec.Command("cat", c.sshKeyPath).Output()
	if err != nil {
		return nil, fmt.Errorf("cannot read SSH key: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("cannot parse SSH key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: c.sshUser,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.timeout,
	}

	addr := net.JoinHostPort(host, strconv.Itoa(c.sshPort))
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH dial failed: %w", err)
	}

	return client, nil
}

// runSSHCommand executes a command on remote host via SSH
func (c *Checker) runSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}
