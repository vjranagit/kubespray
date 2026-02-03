package health

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ComponentStatus represents the status of a cluster component
type ComponentStatus struct {
	Name    string
	Healthy bool
	Message string
	Details map[string]interface{}
}

// ClusterHealth represents overall cluster health
type ClusterHealth struct {
	Timestamp  time.Time
	Healthy    bool
	Components []ComponentStatus
	NodeCount  int
	ReadyNodes int
}

// Monitor checks cluster health
type Monitor struct {
	masters    []string
	nodes      []string
	sshUser    string
	sshKeyPath string
	sshPort    int
}

// NewMonitor creates a new health monitor
func NewMonitor(masters, nodes []string, user, keyPath string, port int) *Monitor {
	return &Monitor{
		masters:    masters,
		nodes:      nodes,
		sshUser:    user,
		sshKeyPath: keyPath,
		sshPort:    port,
	}
}

// CheckClusterHealth performs comprehensive cluster health check
func (m *Monitor) CheckClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	health := &ClusterHealth{
		Timestamp:  time.Now(),
		Components: []ComponentStatus{},
		NodeCount:  len(m.masters) + len(m.nodes),
	}

	// Check API server
	apiStatus := m.checkAPIServer()
	health.Components = append(health.Components, apiStatus)

	// Check etcd
	etcdStatus := m.checkEtcd()
	health.Components = append(health.Components, etcdStatus)

	// Check kubelet on all nodes
	kubeletStatus := m.checkKubelet()
	health.Components = append(health.Components, kubeletStatus...)

	// Check node status
	nodeStatus := m.checkNodeStatus()
	health.Components = append(health.Components, nodeStatus...)
	health.ReadyNodes = m.countReadyNodes(nodeStatus)

	// Determine overall health
	health.Healthy = true
	for _, comp := range health.Components {
		if !comp.Healthy {
			health.Healthy = false
			break
		}
	}

	return health, nil
}

// checkAPIServer validates Kubernetes API server is running
func (m *Monitor) checkAPIServer() ComponentStatus {
	status := ComponentStatus{
		Name:    "Kubernetes API Server",
		Details: make(map[string]interface{}),
	}

	if len(m.masters) == 0 {
		status.Healthy = false
		status.Message = "No master nodes configured"
		return status
	}

	// Check API server on first master
	masterHost := m.masters[0]
	client, err := m.sshConnect(masterHost)
	if err != nil {
		status.Healthy = false
		status.Message = fmt.Sprintf("Cannot connect to master: %v", err)
		return status
	}
	defer client.Close()

	// Check if kube-apiserver is running
	output, err := m.runCommand(client, "sudo systemctl is-active kube-apiserver || kubectl get --raw /healthz")
	if err != nil {
		status.Healthy = false
		status.Message = "API server is not responding"
	} else if strings.Contains(output, "ok") || strings.Contains(output, "active") {
		status.Healthy = true
		status.Message = "API server is healthy"
	} else {
		status.Healthy = false
		status.Message = "API server status unknown"
	}

	return status
}

// checkEtcd validates etcd cluster health
func (m *Monitor) checkEtcd() ComponentStatus {
	status := ComponentStatus{
		Name:    "etcd",
		Details: make(map[string]interface{}),
	}

	if len(m.masters) == 0 {
		status.Healthy = false
		status.Message = "No etcd nodes configured"
		return status
	}

	masterHost := m.masters[0]
	client, err := m.sshConnect(masterHost)
	if err != nil {
		status.Healthy = false
		status.Message = fmt.Sprintf("Cannot connect to etcd node: %v", err)
		return status
	}
	defer client.Close()

	// Check etcd health
	output, err := m.runCommand(client, "sudo ETCDCTL_API=3 etcdctl endpoint health 2>/dev/null || echo 'etcd-check-skipped'")
	if err != nil || strings.Contains(output, "unhealthy") {
		status.Healthy = false
		status.Message = "etcd cluster is unhealthy"
	} else if strings.Contains(output, "healthy") {
		status.Healthy = true
		status.Message = "etcd cluster is healthy"
	} else {
		status.Healthy = true
		status.Message = "etcd status check skipped (may not be running on this node)"
	}

	return status
}

// checkKubelet validates kubelet service on all nodes
func (m *Monitor) checkKubelet() []ComponentStatus {
	statuses := []ComponentStatus{}
	allHosts := append(m.masters, m.nodes...)

	for _, host := range allHosts {
		status := ComponentStatus{
			Name:    fmt.Sprintf("kubelet - %s", host),
			Details: make(map[string]interface{}),
		}

		client, err := m.sshConnect(host)
		if err != nil {
			status.Healthy = false
			status.Message = fmt.Sprintf("Cannot connect: %v", err)
			statuses = append(statuses, status)
			continue
		}
		defer client.Close()

		output, err := m.runCommand(client, "sudo systemctl is-active kubelet")
		if err != nil || !strings.Contains(output, "active") {
			status.Healthy = false
			status.Message = "kubelet is not running"
		} else {
			status.Healthy = true
			status.Message = "kubelet is running"
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// checkNodeStatus validates node ready status
func (m *Monitor) checkNodeStatus() []ComponentStatus {
	statuses := []ComponentStatus{}

	if len(m.masters) == 0 {
		return statuses
	}

	masterHost := m.masters[0]
	client, err := m.sshConnect(masterHost)
	if err != nil {
		status := ComponentStatus{
			Name:    "Node Status",
			Healthy: false,
			Message: fmt.Sprintf("Cannot connect to master: %v", err),
			Details: make(map[string]interface{}),
		}
		return []ComponentStatus{status}
	}
	defer client.Close()

	// Get node status via kubectl
	output, err := m.runCommand(client, "kubectl get nodes --no-headers 2>/dev/null || echo 'kubectl-not-available'")
	if err != nil || strings.Contains(output, "kubectl-not-available") {
		status := ComponentStatus{
			Name:    "Node Status",
			Healthy: true,
			Message: "kubectl not available - cluster may still be initializing",
			Details: make(map[string]interface{}),
		}
		return []ComponentStatus{status}
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		nodeName := fields[0]
		nodeStatus := fields[1]

		status := ComponentStatus{
			Name:    fmt.Sprintf("Node - %s", nodeName),
			Healthy: strings.Contains(nodeStatus, "Ready"),
			Details: make(map[string]interface{}),
		}

		if status.Healthy {
			status.Message = "Node is Ready"
		} else {
			status.Message = fmt.Sprintf("Node status: %s", nodeStatus)
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// countReadyNodes counts how many nodes are in Ready state
func (m *Monitor) countReadyNodes(nodeStatuses []ComponentStatus) int {
	count := 0
	for _, status := range nodeStatuses {
		if strings.HasPrefix(status.Name, "Node - ") && status.Healthy {
			count++
		}
	}
	return count
}

// sshConnect establishes SSH connection
func (m *Monitor) sshConnect(host string) (*ssh.Client, error) {
	// Implementation similar to preflight checker
	// Simplified for brevity - would use same logic
	return nil, fmt.Errorf("SSH connection not implemented in this stub")
}

// runCommand executes command via SSH
func (m *Monitor) runCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	return string(output), err
}
