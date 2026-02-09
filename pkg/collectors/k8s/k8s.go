// Package k8s provides a Kubernetes cluster status collector for prompt-pulse.
// It queries the K8s API via client-go to gather node, pod, deployment, and
// namespace information across one or more kubeconfig contexts.
package k8s

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Default values for collector configuration.
const (
	defaultInterval = 15 * time.Second
)

// ---------- Configuration ----------

// Config holds the configuration for the Kubernetes collector.
type Config struct {
	// Interval is the collection polling interval. Defaults to 15s.
	Interval time.Duration

	// Kubeconfig is the path to a kubeconfig file. If empty, the default
	// loading rules apply (KUBECONFIG env, ~/.kube/config, in-cluster).
	Kubeconfig string

	// Contexts lists specific kubeconfig contexts to monitor. If empty,
	// only the current context is used.
	Contexts []string

	// Namespaces restricts collection to specific namespaces. If empty,
	// all namespaces are queried.
	Namespaces []string
}

// ---------- Result types ----------

// ClusterStatus is the top-level data returned by Collect.
type ClusterStatus struct {
	Clusters  []ClusterInfo `json:"clusters"`
	Timestamp time.Time     `json:"timestamp"`
}

// ClusterInfo holds status information for a single Kubernetes context.
type ClusterInfo struct {
	Context     string          `json:"context"`
	Connected   bool            `json:"connected"`
	Error       string          `json:"error,omitempty"`
	Nodes       []NodeInfo      `json:"nodes,omitempty"`
	Namespaces  []NamespaceInfo `json:"namespaces,omitempty"`
	TotalPods   int             `json:"total_pods"`
	RunningPods int             `json:"running_pods"`
	PendingPods int             `json:"pending_pods"`
	FailedPods  int             `json:"failed_pods"`
}

// NodeInfo holds status and resource information for a single node.
type NodeInfo struct {
	Name        string   `json:"name"`
	Ready       bool     `json:"ready"`
	Roles       []string `json:"roles,omitempty"`
	CPUCapacity string   `json:"cpu_capacity"`
	CPURequests string   `json:"cpu_requests"`
	CPULimits   string   `json:"cpu_limits"`
	MemCapacity string   `json:"mem_capacity"`
	MemRequests string   `json:"mem_requests"`
	MemLimits   string   `json:"mem_limits"`
	PodCount    int      `json:"pod_count"`
	Conditions  []string `json:"conditions,omitempty"`
}

// NamespaceInfo holds pod and deployment information for a single namespace.
type NamespaceInfo struct {
	Name        string           `json:"name"`
	PodCounts   PodCounts        `json:"pod_counts"`
	Deployments []DeploymentInfo `json:"deployments,omitempty"`
}

// PodCounts tracks pod phase counts within a namespace.
type PodCounts struct {
	Total     int `json:"total"`
	Running   int `json:"running"`
	Pending   int `json:"pending"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Unknown   int `json:"unknown"`
}

// DeploymentInfo holds replica and condition info for a single deployment.
type DeploymentInfo struct {
	Name              string   `json:"name"`
	Replicas          int32    `json:"replicas"`
	ReadyReplicas     int32    `json:"ready_replicas"`
	UpdatedReplicas   int32    `json:"updated_replicas"`
	AvailableReplicas int32    `json:"available_replicas"`
	Conditions        []string `json:"conditions,omitempty"`
}

// ---------- K8sClient interface ----------

// K8sClient abstracts Kubernetes API calls for testability.
type K8sClient interface {
	ListNodes(ctx context.Context) ([]corev1.Node, error)
	ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error)
	ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error)
	ListNamespaces(ctx context.Context) ([]corev1.Namespace, error)
}

// realClient wraps a kubernetes.Clientset to implement K8sClient.
type realClient struct {
	cs *kubernetes.Clientset
}

func (r *realClient) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	list, err := r.cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *realClient) ListPods(ctx context.Context, namespace string) ([]corev1.Pod, error) {
	list, err := r.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *realClient) ListDeployments(ctx context.Context, namespace string) ([]appsv1.Deployment, error) {
	list, err := r.cs.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

func (r *realClient) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	list, err := r.cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}

// ---------- clientFactory ----------

// clientFactory creates K8sClient instances for a given kubeconfig context.
// It is a function type so tests can inject mock factory implementations.
type clientFactory func(kubeconfig, context string) (K8sClient, error)

// defaultClientFactory builds a real K8sClient from a kubeconfig path and context.
func defaultClientFactory(kubeconfig, ctxName string) (K8sClient, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfig != "" {
		rules.ExplicitPath = kubeconfig
	}
	overrides := &clientcmd.ConfigOverrides{}
	if ctxName != "" {
		overrides.CurrentContext = ctxName
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("build client config: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return &realClient{cs: cs}, nil
}

// ---------- Collector ----------

// Collector implements the pkg/collectors.Collector interface for Kubernetes.
type Collector struct {
	cfg     Config
	factory clientFactory

	mu      sync.RWMutex
	healthy bool
}

// New creates a Collector with the given configuration.
func New(cfg Config) *Collector {
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	return &Collector{
		cfg:     cfg,
		factory: defaultClientFactory,
		healthy: true,
	}
}

// newWithFactory creates a Collector with a custom client factory (for tests).
func newWithFactory(cfg Config, factory clientFactory) *Collector {
	c := New(cfg)
	c.factory = factory
	return c
}

// Name returns the collector identifier.
func (c *Collector) Name() string { return "k8s" }

// Interval returns the configured polling interval.
func (c *Collector) Interval() time.Duration { return c.cfg.Interval }

// Healthy returns true if the last collection succeeded.
func (c *Collector) Healthy() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.healthy
}

func (c *Collector) setHealthy(h bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.healthy = h
}

// Collect gathers Kubernetes cluster status from all configured contexts.
// On success, Healthy() returns true. On total failure, Healthy() returns false
// but a partial ClusterStatus with error details is still returned (not a Go error).
func (c *Collector) Collect(ctx context.Context) (interface{}, error) {
	contexts := c.cfg.Contexts
	if len(contexts) == 0 {
		// Use the current/default context (empty string means default).
		contexts = []string{""}
	}

	status := &ClusterStatus{
		Clusters:  make([]ClusterInfo, 0, len(contexts)),
		Timestamp: time.Now(),
	}

	anyConnected := false

	for _, ctxName := range contexts {
		info := c.collectContext(ctx, ctxName)
		status.Clusters = append(status.Clusters, info)
		if info.Connected {
			anyConnected = true
		}
	}

	c.setHealthy(anyConnected || len(contexts) == 0)

	return status, nil
}

// collectContext gathers data for a single kubeconfig context.
func (c *Collector) collectContext(ctx context.Context, ctxName string) ClusterInfo {
	info := ClusterInfo{
		Context: ctxName,
	}

	client, err := c.factory(c.cfg.Kubeconfig, ctxName)
	if err != nil {
		info.Error = err.Error()
		return info
	}

	// Fetch nodes.
	nodes, err := client.ListNodes(ctx)
	if err != nil {
		info.Error = fmt.Sprintf("list nodes: %v", err)
		return info
	}

	// We have connectivity if we can list nodes.
	info.Connected = true

	// Determine which namespaces to query.
	namespacesToQuery, err := c.resolveNamespaces(ctx, client)
	if err != nil {
		// Non-fatal: we already proved connectivity via ListNodes.
		info.Error = fmt.Sprintf("list namespaces: %v", err)
	}

	// Fetch all pods across target namespaces.
	allPods, podsByNs := c.collectPods(ctx, client, namespacesToQuery)

	// Fetch all deployments across target namespaces.
	deploysByNs := c.collectDeployments(ctx, client, namespacesToQuery)

	// Build node info (with pod counts per node).
	podCountsByNode := countPodsByNode(allPods)
	for i := range nodes {
		ni := buildNodeInfo(&nodes[i], podCountsByNode, allPods)
		info.Nodes = append(info.Nodes, ni)
	}

	// Build namespace info.
	for _, ns := range namespacesToQuery {
		nsInfo := NamespaceInfo{
			Name:      ns,
			PodCounts: countPodPhases(podsByNs[ns]),
		}
		if deps, ok := deploysByNs[ns]; ok {
			for i := range deps {
				nsInfo.Deployments = append(nsInfo.Deployments, buildDeploymentInfo(&deps[i]))
			}
		}
		info.Namespaces = append(info.Namespaces, nsInfo)
	}

	// Aggregate pod counts.
	info.TotalPods, info.RunningPods, info.PendingPods, info.FailedPods = aggregatePodCounts(allPods)

	return info
}

// resolveNamespaces returns the list of namespaces to query.
func (c *Collector) resolveNamespaces(ctx context.Context, client K8sClient) ([]string, error) {
	if len(c.cfg.Namespaces) > 0 {
		return c.cfg.Namespaces, nil
	}
	nsList, err := client.ListNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(nsList))
	for _, ns := range nsList {
		names = append(names, ns.Name)
	}
	return names, nil
}

// collectPods fetches pods from all target namespaces. Returns a flat list
// of all pods and a per-namespace map.
func (c *Collector) collectPods(ctx context.Context, client K8sClient, namespaces []string) ([]corev1.Pod, map[string][]corev1.Pod) {
	var all []corev1.Pod
	byNs := make(map[string][]corev1.Pod, len(namespaces))

	if len(c.cfg.Namespaces) > 0 {
		// Fetch per-namespace when filtered.
		for _, ns := range namespaces {
			pods, err := client.ListPods(ctx, ns)
			if err != nil {
				continue
			}
			byNs[ns] = pods
			all = append(all, pods...)
		}
	} else {
		// Fetch all namespaces at once (empty string = all).
		pods, err := client.ListPods(ctx, "")
		if err != nil {
			return nil, byNs
		}
		all = pods
		for i := range pods {
			ns := pods[i].Namespace
			byNs[ns] = append(byNs[ns], pods[i])
		}
	}
	return all, byNs
}

// collectDeployments fetches deployments from all target namespaces.
func (c *Collector) collectDeployments(ctx context.Context, client K8sClient, namespaces []string) map[string][]appsv1.Deployment {
	byNs := make(map[string][]appsv1.Deployment, len(namespaces))

	if len(c.cfg.Namespaces) > 0 {
		for _, ns := range namespaces {
			deps, err := client.ListDeployments(ctx, ns)
			if err != nil {
				continue
			}
			byNs[ns] = deps
		}
	} else {
		deps, err := client.ListDeployments(ctx, "")
		if err != nil {
			return byNs
		}
		for i := range deps {
			ns := deps[i].Namespace
			byNs[ns] = append(byNs[ns], deps[i])
		}
	}
	return byNs
}

// ---------- Node helpers ----------

// buildNodeInfo constructs a NodeInfo from a corev1.Node and pod data.
func buildNodeInfo(node *corev1.Node, podCountsByNode map[string]int, allPods []corev1.Pod) NodeInfo {
	ni := NodeInfo{
		Name:     node.Name,
		Ready:    isNodeReady(node),
		Roles:    extractRoles(node),
		PodCount: podCountsByNode[node.Name],
	}

	// Resource capacity.
	if cap := node.Status.Capacity; cap != nil {
		if cpu, ok := cap[corev1.ResourceCPU]; ok {
			ni.CPUCapacity = cpu.String()
		}
		if mem, ok := cap[corev1.ResourceMemory]; ok {
			ni.MemCapacity = mem.String()
		}
	}

	// Resource requests and limits from pods on this node.
	var cpuReq, cpuLim, memReq, memLim int64
	for i := range allPods {
		if allPods[i].Spec.NodeName != node.Name {
			continue
		}
		for j := range allPods[i].Spec.Containers {
			c := &allPods[i].Spec.Containers[j]
			if c.Resources.Requests != nil {
				if v, ok := c.Resources.Requests[corev1.ResourceCPU]; ok {
					cpuReq += v.MilliValue()
				}
				if v, ok := c.Resources.Requests[corev1.ResourceMemory]; ok {
					memReq += v.Value()
				}
			}
			if c.Resources.Limits != nil {
				if v, ok := c.Resources.Limits[corev1.ResourceCPU]; ok {
					cpuLim += v.MilliValue()
				}
				if v, ok := c.Resources.Limits[corev1.ResourceMemory]; ok {
					memLim += v.Value()
				}
			}
		}
	}
	ni.CPURequests = fmt.Sprintf("%dm", cpuReq)
	ni.CPULimits = fmt.Sprintf("%dm", cpuLim)
	ni.MemRequests = fmt.Sprintf("%d", memReq)
	ni.MemLimits = fmt.Sprintf("%d", memLim)

	// Non-Ready conditions.
	ni.Conditions = extractNonReadyConditions(node)

	return ni
}

// isNodeReady checks whether a node has a Ready condition set to True.
func isNodeReady(node *corev1.Node) bool {
	if node == nil {
		return false
	}
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

// extractRoles returns the roles of a node based on standard labels.
func extractRoles(node *corev1.Node) []string {
	if node == nil || node.Labels == nil {
		return nil
	}
	var roles []string
	for label := range node.Labels {
		const prefix = "node-role.kubernetes.io/"
		if strings.HasPrefix(label, prefix) {
			role := strings.TrimPrefix(label, prefix)
			if role != "" {
				roles = append(roles, role)
			}
		}
	}
	if role, ok := node.Labels["kubernetes.io/role"]; ok && role != "" {
		// Avoid duplicates.
		found := false
		for _, r := range roles {
			if r == role {
				found = true
				break
			}
		}
		if !found {
			roles = append(roles, role)
		}
	}
	return roles
}

// extractNonReadyConditions returns condition type names for conditions that
// are not Ready and have status True (indicating a problem).
func extractNonReadyConditions(node *corev1.Node) []string {
	if node == nil {
		return nil
	}
	var conds []string
	for _, cond := range node.Status.Conditions {
		if cond.Type != corev1.NodeReady && cond.Status == corev1.ConditionTrue {
			conds = append(conds, string(cond.Type))
		}
	}
	return conds
}

// ---------- Pod helpers ----------

// countPodsByNode maps node name to the number of pods scheduled on it.
func countPodsByNode(pods []corev1.Pod) map[string]int {
	counts := make(map[string]int)
	for i := range pods {
		if pods[i].Spec.NodeName != "" {
			counts[pods[i].Spec.NodeName]++
		}
	}
	return counts
}

// countPodPhases aggregates pod counts by phase within a namespace.
func countPodPhases(pods []corev1.Pod) PodCounts {
	var pc PodCounts
	pc.Total = len(pods)
	for i := range pods {
		switch pods[i].Status.Phase {
		case corev1.PodRunning:
			pc.Running++
		case corev1.PodPending:
			pc.Pending++
		case corev1.PodSucceeded:
			pc.Succeeded++
		case corev1.PodFailed:
			pc.Failed++
		default:
			pc.Unknown++
		}
	}
	return pc
}

// aggregatePodCounts returns total, running, pending, and failed pod counts.
func aggregatePodCounts(pods []corev1.Pod) (total, running, pending, failed int) {
	total = len(pods)
	for i := range pods {
		switch pods[i].Status.Phase {
		case corev1.PodRunning:
			running++
		case corev1.PodPending:
			pending++
		case corev1.PodFailed:
			failed++
		}
	}
	return
}

// ---------- Deployment helpers ----------

// buildDeploymentInfo constructs a DeploymentInfo from an appsv1.Deployment.
func buildDeploymentInfo(dep *appsv1.Deployment) DeploymentInfo {
	di := DeploymentInfo{
		Name: dep.Name,
	}
	if dep.Spec.Replicas != nil {
		di.Replicas = *dep.Spec.Replicas
	}
	di.ReadyReplicas = dep.Status.ReadyReplicas
	di.UpdatedReplicas = dep.Status.UpdatedReplicas
	di.AvailableReplicas = dep.Status.AvailableReplicas

	for _, cond := range dep.Status.Conditions {
		di.Conditions = append(di.Conditions, fmt.Sprintf("%s=%s", cond.Type, cond.Status))
	}

	return di
}
