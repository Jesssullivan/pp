package k8s

import (
	"context"
	"errors"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ---------- Mock K8sClient ----------

// mockClient implements K8sClient with configurable return values.
type mockClient struct {
	nodes       []corev1.Node
	nodesErr    error
	pods        map[string][]corev1.Pod // namespace -> pods (empty key = all)
	podsErr     error
	deployments map[string][]appsv1.Deployment
	depsErr     error
	namespaces  []corev1.Namespace
	nsErr       error
}

func (m *mockClient) ListNodes(_ context.Context) ([]corev1.Node, error) {
	return m.nodes, m.nodesErr
}

func (m *mockClient) ListPods(_ context.Context, namespace string) ([]corev1.Pod, error) {
	if m.podsErr != nil {
		return nil, m.podsErr
	}
	if pods, ok := m.pods[namespace]; ok {
		return pods, nil
	}
	return nil, nil
}

func (m *mockClient) ListDeployments(_ context.Context, namespace string) ([]appsv1.Deployment, error) {
	if m.depsErr != nil {
		return nil, m.depsErr
	}
	if deps, ok := m.deployments[namespace]; ok {
		return deps, nil
	}
	return nil, nil
}

func (m *mockClient) ListNamespaces(_ context.Context) ([]corev1.Namespace, error) {
	return m.namespaces, m.nsErr
}

// ---------- Helper builders ----------

func makeNode(name string, ready bool, labels map[string]string, cpuCap, memCap string, extraConditions ...corev1.NodeCondition) corev1.Node {
	status := corev1.ConditionFalse
	if ready {
		status = corev1.ConditionTrue
	}

	conditions := []corev1.NodeCondition{
		{Type: corev1.NodeReady, Status: status},
	}
	conditions = append(conditions, extraConditions...)

	n := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Conditions: conditions,
			Capacity:   corev1.ResourceList{},
		},
	}
	if cpuCap != "" {
		n.Status.Capacity[corev1.ResourceCPU] = resource.MustParse(cpuCap)
	}
	if memCap != "" {
		n.Status.Capacity[corev1.ResourceMemory] = resource.MustParse(memCap)
	}
	return n
}

func makePod(name, namespace, nodeName string, phase corev1.PodPhase, cpuReq, memReq string) corev1.Pod {
	p := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
	if cpuReq != "" || memReq != "" {
		container := corev1.Container{
			Name: "main",
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
				Limits:   corev1.ResourceList{},
			},
		}
		if cpuReq != "" {
			container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(cpuReq)
			container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(cpuReq)
		}
		if memReq != "" {
			container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(memReq)
			container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(memReq)
		}
		p.Spec.Containers = []corev1.Container{container}
	}
	return p
}

func makeDeployment(name, namespace string, replicas, ready, updated, available int32, conditions ...appsv1.DeploymentCondition) appsv1.Deployment {
	return appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     ready,
			UpdatedReplicas:   updated,
			AvailableReplicas: available,
			Conditions:        conditions,
		},
	}
}

func makeNamespace(name string) corev1.Namespace {
	return corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
}

func int32Ptr(v int32) *int32 { return &v }

// mockFactory returns a clientFactory that ignores kubeconfig/context and
// always returns the provided mockClient.
func mockFactory(client K8sClient) clientFactory {
	return func(_, _ string) (K8sClient, error) {
		return client, nil
	}
}

// errorFactory returns a clientFactory that always returns an error.
func errorFactory(err error) clientFactory {
	return func(_, _ string) (K8sClient, error) {
		return nil, err
	}
}

// contextFactory returns a clientFactory that maps context names to clients.
func contextFactory(clients map[string]K8sClient) clientFactory {
	return func(_, ctxName string) (K8sClient, error) {
		if c, ok := clients[ctxName]; ok {
			return c, nil
		}
		return nil, errors.New("context not found")
	}
}

// ---------- Tests ----------

func TestName(t *testing.T) {
	c := New(Config{})
	if got := c.Name(); got != "k8s" {
		t.Errorf("Name() = %q, want %q", got, "k8s")
	}
}

func TestInterval_Default(t *testing.T) {
	c := New(Config{})
	if got := c.Interval(); got != 15*time.Second {
		t.Errorf("Interval() = %v, want %v", got, 15*time.Second)
	}
}

func TestInterval_Custom(t *testing.T) {
	c := New(Config{Interval: 30 * time.Second})
	if got := c.Interval(); got != 30*time.Second {
		t.Errorf("Interval() = %v, want %v", got, 30*time.Second)
	}
}

func TestDefaultConfig(t *testing.T) {
	c := New(Config{})
	if c.cfg.Interval != defaultInterval {
		t.Errorf("default interval = %v, want %v", c.cfg.Interval, defaultInterval)
	}
	if c.cfg.Kubeconfig != "" {
		t.Errorf("default kubeconfig = %q, want empty", c.cfg.Kubeconfig)
	}
	if len(c.cfg.Contexts) != 0 {
		t.Errorf("default contexts = %v, want empty", c.cfg.Contexts)
	}
	if len(c.cfg.Namespaces) != 0 {
		t.Errorf("default namespaces = %v, want empty", c.cfg.Namespaces)
	}
	if !c.Healthy() {
		t.Error("new collector should be healthy")
	}
}

func TestCollect_TwoNodes_MixedReadiness(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, map[string]string{
				"node-role.kubernetes.io/control-plane": "",
			}, "4", "8Gi"),
			makeNode("node-2", false, map[string]string{
				"node-role.kubernetes.io/worker": "",
			}, "8", "16Gi",
				corev1.NodeCondition{Type: corev1.NodeDiskPressure, Status: corev1.ConditionTrue},
			),
		},
		pods: map[string][]corev1.Pod{
			"": {
				makePod("pod-1", "default", "node-1", corev1.PodRunning, "100m", "128Mi"),
				makePod("pod-2", "default", "node-1", corev1.PodRunning, "200m", "256Mi"),
				makePod("pod-3", "kube-system", "node-2", corev1.PodPending, "", ""),
			},
		},
		namespaces: []corev1.Namespace{
			makeNamespace("default"),
			makeNamespace("kube-system"),
		},
		deployments: map[string][]appsv1.Deployment{
			"": {
				makeDeployment("nginx", "default", 3, 2, 3, 2),
			},
		},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status, ok := result.(*ClusterStatus)
	if !ok {
		t.Fatalf("Collect() returned %T, want *ClusterStatus", result)
	}

	if len(status.Clusters) != 1 {
		t.Fatalf("len(Clusters) = %d, want 1", len(status.Clusters))
	}

	cluster := status.Clusters[0]
	if !cluster.Connected {
		t.Error("cluster should be connected")
	}

	// Node checks.
	if len(cluster.Nodes) != 2 {
		t.Fatalf("len(Nodes) = %d, want 2", len(cluster.Nodes))
	}

	n1 := cluster.Nodes[0]
	if n1.Name != "node-1" {
		t.Errorf("Nodes[0].Name = %q, want %q", n1.Name, "node-1")
	}
	if !n1.Ready {
		t.Error("node-1 should be ready")
	}
	if len(n1.Roles) != 1 || n1.Roles[0] != "control-plane" {
		t.Errorf("node-1 roles = %v, want [control-plane]", n1.Roles)
	}
	if n1.PodCount != 2 {
		t.Errorf("node-1 PodCount = %d, want 2", n1.PodCount)
	}

	n2 := cluster.Nodes[1]
	if n2.Name != "node-2" {
		t.Errorf("Nodes[1].Name = %q, want %q", n2.Name, "node-2")
	}
	if n2.Ready {
		t.Error("node-2 should NOT be ready")
	}
	if n2.PodCount != 1 {
		t.Errorf("node-2 PodCount = %d, want 1", n2.PodCount)
	}

	// Check non-Ready conditions on node-2.
	foundDiskPressure := false
	for _, cond := range n2.Conditions {
		if cond == "DiskPressure" {
			foundDiskPressure = true
		}
	}
	if !foundDiskPressure {
		t.Errorf("node-2 conditions = %v, want DiskPressure present", n2.Conditions)
	}

	// Pod counts.
	if cluster.TotalPods != 3 {
		t.Errorf("TotalPods = %d, want 3", cluster.TotalPods)
	}
	if cluster.RunningPods != 2 {
		t.Errorf("RunningPods = %d, want 2", cluster.RunningPods)
	}
	if cluster.PendingPods != 1 {
		t.Errorf("PendingPods = %d, want 1", cluster.PendingPods)
	}

	if !c.Healthy() {
		t.Error("collector should be healthy after successful collect")
	}
}

func TestCollect_PodPhases(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods: map[string][]corev1.Pod{
			"": {
				makePod("running-1", "default", "node-1", corev1.PodRunning, "", ""),
				makePod("running-2", "default", "node-1", corev1.PodRunning, "", ""),
				makePod("pending-1", "default", "node-1", corev1.PodPending, "", ""),
				makePod("failed-1", "default", "node-1", corev1.PodFailed, "", ""),
				makePod("succeeded-1", "default", "node-1", corev1.PodSucceeded, "", ""),
				makePod("unknown-1", "default", "node-1", corev1.PodUnknown, "", ""),
			},
		},
		namespaces: []corev1.Namespace{
			makeNamespace("default"),
		},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]

	if cluster.TotalPods != 6 {
		t.Errorf("TotalPods = %d, want 6", cluster.TotalPods)
	}
	if cluster.RunningPods != 2 {
		t.Errorf("RunningPods = %d, want 2", cluster.RunningPods)
	}
	if cluster.PendingPods != 1 {
		t.Errorf("PendingPods = %d, want 1", cluster.PendingPods)
	}
	if cluster.FailedPods != 1 {
		t.Errorf("FailedPods = %d, want 1", cluster.FailedPods)
	}

	// Check namespace-level pod counts.
	if len(cluster.Namespaces) != 1 {
		t.Fatalf("len(Namespaces) = %d, want 1", len(cluster.Namespaces))
	}
	ns := cluster.Namespaces[0]
	if ns.PodCounts.Total != 6 {
		t.Errorf("ns.PodCounts.Total = %d, want 6", ns.PodCounts.Total)
	}
	if ns.PodCounts.Running != 2 {
		t.Errorf("ns.PodCounts.Running = %d, want 2", ns.PodCounts.Running)
	}
	if ns.PodCounts.Pending != 1 {
		t.Errorf("ns.PodCounts.Pending = %d, want 1", ns.PodCounts.Pending)
	}
	if ns.PodCounts.Succeeded != 1 {
		t.Errorf("ns.PodCounts.Succeeded = %d, want 1", ns.PodCounts.Succeeded)
	}
	if ns.PodCounts.Failed != 1 {
		t.Errorf("ns.PodCounts.Failed = %d, want 1", ns.PodCounts.Failed)
	}
	if ns.PodCounts.Unknown != 1 {
		t.Errorf("ns.PodCounts.Unknown = %d, want 1", ns.PodCounts.Unknown)
	}
}

func TestCollect_PodCountsAggregateByNamespace(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods: map[string][]corev1.Pod{
			"": {
				makePod("pod-1", "ns-a", "node-1", corev1.PodRunning, "", ""),
				makePod("pod-2", "ns-a", "node-1", corev1.PodRunning, "", ""),
				makePod("pod-3", "ns-b", "node-1", corev1.PodPending, "", ""),
				makePod("pod-4", "ns-b", "node-1", corev1.PodFailed, "", ""),
				makePod("pod-5", "ns-b", "node-1", corev1.PodRunning, "", ""),
			},
		},
		namespaces: []corev1.Namespace{
			makeNamespace("ns-a"),
			makeNamespace("ns-b"),
		},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]

	if len(cluster.Namespaces) != 2 {
		t.Fatalf("len(Namespaces) = %d, want 2", len(cluster.Namespaces))
	}

	// Find ns-a and ns-b by name.
	nsMap := make(map[string]NamespaceInfo)
	for _, ns := range cluster.Namespaces {
		nsMap[ns.Name] = ns
	}

	nsA := nsMap["ns-a"]
	if nsA.PodCounts.Total != 2 {
		t.Errorf("ns-a Total = %d, want 2", nsA.PodCounts.Total)
	}
	if nsA.PodCounts.Running != 2 {
		t.Errorf("ns-a Running = %d, want 2", nsA.PodCounts.Running)
	}

	nsB := nsMap["ns-b"]
	if nsB.PodCounts.Total != 3 {
		t.Errorf("ns-b Total = %d, want 3", nsB.PodCounts.Total)
	}
	if nsB.PodCounts.Running != 1 {
		t.Errorf("ns-b Running = %d, want 1", nsB.PodCounts.Running)
	}
	if nsB.PodCounts.Pending != 1 {
		t.Errorf("ns-b Pending = %d, want 1", nsB.PodCounts.Pending)
	}
	if nsB.PodCounts.Failed != 1 {
		t.Errorf("ns-b Failed = %d, want 1", nsB.PodCounts.Failed)
	}
}

func TestCollect_Deployment(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods: map[string][]corev1.Pod{
			"": {},
		},
		namespaces: []corev1.Namespace{
			makeNamespace("default"),
		},
		deployments: map[string][]appsv1.Deployment{
			"": {
				makeDeployment("web", "default", 5, 3, 4, 3,
					appsv1.DeploymentCondition{
						Type:   appsv1.DeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
					appsv1.DeploymentCondition{
						Type:   appsv1.DeploymentProgressing,
						Status: corev1.ConditionTrue,
					},
				),
			},
		},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]

	if len(cluster.Namespaces) != 1 {
		t.Fatalf("len(Namespaces) = %d, want 1", len(cluster.Namespaces))
	}
	ns := cluster.Namespaces[0]
	if len(ns.Deployments) != 1 {
		t.Fatalf("len(Deployments) = %d, want 1", len(ns.Deployments))
	}

	dep := ns.Deployments[0]
	if dep.Name != "web" {
		t.Errorf("deployment name = %q, want %q", dep.Name, "web")
	}
	if dep.Replicas != 5 {
		t.Errorf("Replicas = %d, want 5", dep.Replicas)
	}
	if dep.ReadyReplicas != 3 {
		t.Errorf("ReadyReplicas = %d, want 3", dep.ReadyReplicas)
	}
	if dep.UpdatedReplicas != 4 {
		t.Errorf("UpdatedReplicas = %d, want 4", dep.UpdatedReplicas)
	}
	if dep.AvailableReplicas != 3 {
		t.Errorf("AvailableReplicas = %d, want 3", dep.AvailableReplicas)
	}
	if len(dep.Conditions) != 2 {
		t.Errorf("len(Conditions) = %d, want 2", len(dep.Conditions))
	}
}

func TestCollect_NodeResourceCapacity(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods: map[string][]corev1.Pod{
			"": {
				makePod("pod-1", "default", "node-1", corev1.PodRunning, "250m", "512Mi"),
			},
		},
		namespaces: []corev1.Namespace{
			makeNamespace("default"),
		},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	node := status.Clusters[0].Nodes[0]

	if node.CPUCapacity != "4" {
		t.Errorf("CPUCapacity = %q, want %q", node.CPUCapacity, "4")
	}
	if node.MemCapacity != "8Gi" {
		t.Errorf("MemCapacity = %q, want %q", node.MemCapacity, "8Gi")
	}
	if node.CPURequests != "250m" {
		t.Errorf("CPURequests = %q, want %q", node.CPURequests, "250m")
	}
	if node.CPULimits != "250m" {
		t.Errorf("CPULimits = %q, want %q", node.CPULimits, "250m")
	}
	// 512Mi = 536870912 bytes.
	if node.MemRequests != "536870912" {
		t.Errorf("MemRequests = %q, want %q", node.MemRequests, "536870912")
	}
}

func TestCollect_ClientError_Unhealthy(t *testing.T) {
	c := newWithFactory(Config{}, errorFactory(errors.New("connection refused")))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() should not return Go error, got: %v", err)
	}

	status := result.(*ClusterStatus)
	if len(status.Clusters) != 1 {
		t.Fatalf("len(Clusters) = %d, want 1", len(status.Clusters))
	}

	cluster := status.Clusters[0]
	if cluster.Connected {
		t.Error("cluster should NOT be connected on client error")
	}
	if cluster.Error == "" {
		t.Error("cluster.Error should be set")
	}

	if c.Healthy() {
		t.Error("collector should be unhealthy after connection failure")
	}
}

func TestCollect_ListNodesError_Unhealthy(t *testing.T) {
	mock := &mockClient{
		nodesErr: errors.New("forbidden"),
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() should not return Go error, got: %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]
	if cluster.Connected {
		t.Error("cluster should NOT be connected when ListNodes fails")
	}
	if cluster.Error == "" {
		t.Error("cluster.Error should be set")
	}

	if c.Healthy() {
		t.Error("collector should be unhealthy when ListNodes fails")
	}
}

func TestCollect_SuccessThenError_HealthyToggle(t *testing.T) {
	successMock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	c := newWithFactory(Config{}, mockFactory(successMock))

	// First collect succeeds.
	_, _ = c.Collect(context.Background())
	if !c.Healthy() {
		t.Error("should be healthy after success")
	}

	// Switch to error factory.
	c.factory = errorFactory(errors.New("timeout"))

	_, _ = c.Collect(context.Background())
	if c.Healthy() {
		t.Error("should be unhealthy after failure")
	}
}

func TestCollect_EmptyCluster(t *testing.T) {
	mock := &mockClient{
		nodes:      []corev1.Node{},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]

	if !cluster.Connected {
		t.Error("empty cluster should still be connected")
	}
	if len(cluster.Nodes) != 0 {
		t.Errorf("len(Nodes) = %d, want 0", len(cluster.Nodes))
	}
	if cluster.TotalPods != 0 {
		t.Errorf("TotalPods = %d, want 0", cluster.TotalPods)
	}
	if len(cluster.Namespaces) != 0 {
		t.Errorf("len(Namespaces) = %d, want 0", len(cluster.Namespaces))
	}

	if !c.Healthy() {
		t.Error("should be healthy for empty cluster")
	}
}

func TestCollect_NamespaceFiltering(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods: map[string][]corev1.Pod{
			"app": {
				makePod("pod-1", "app", "node-1", corev1.PodRunning, "", ""),
				makePod("pod-2", "app", "node-1", corev1.PodRunning, "", ""),
			},
			"monitoring": {
				makePod("pod-3", "monitoring", "node-1", corev1.PodRunning, "", ""),
			},
		},
		deployments: map[string][]appsv1.Deployment{
			"app": {
				makeDeployment("web", "app", 2, 2, 2, 2),
			},
			"monitoring": {
				makeDeployment("prom", "monitoring", 1, 1, 1, 1),
			},
		},
		// ListNamespaces should NOT be called when Namespaces is set.
		nsErr: errors.New("should not be called"),
	}

	c := newWithFactory(Config{
		Namespaces: []string{"app", "monitoring"},
	}, mockFactory(mock))

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]

	if !cluster.Connected {
		t.Error("cluster should be connected")
	}

	if len(cluster.Namespaces) != 2 {
		t.Fatalf("len(Namespaces) = %d, want 2", len(cluster.Namespaces))
	}

	nsMap := make(map[string]NamespaceInfo)
	for _, ns := range cluster.Namespaces {
		nsMap[ns.Name] = ns
	}

	app := nsMap["app"]
	if app.PodCounts.Total != 2 {
		t.Errorf("app Total = %d, want 2", app.PodCounts.Total)
	}
	if len(app.Deployments) != 1 {
		t.Errorf("app deployments = %d, want 1", len(app.Deployments))
	}

	mon := nsMap["monitoring"]
	if mon.PodCounts.Total != 1 {
		t.Errorf("monitoring Total = %d, want 1", mon.PodCounts.Total)
	}
	if len(mon.Deployments) != 1 {
		t.Errorf("monitoring deployments = %d, want 1", len(mon.Deployments))
	}

	// Total pods should be 3 (aggregated from namespace-filtered queries).
	if cluster.TotalPods != 3 {
		t.Errorf("TotalPods = %d, want 3", cluster.TotalPods)
	}
}

func TestCollect_MultipleContexts(t *testing.T) {
	prodMock := &mockClient{
		nodes: []corev1.Node{
			makeNode("prod-node-1", true, nil, "8", "32Gi"),
		},
		pods: map[string][]corev1.Pod{
			"": {
				makePod("pod-1", "default", "prod-node-1", corev1.PodRunning, "", ""),
			},
		},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	stagMock := &mockClient{
		nodes: []corev1.Node{
			makeNode("stag-node-1", true, nil, "4", "16Gi"),
			makeNode("stag-node-2", true, nil, "4", "16Gi"),
		},
		pods: map[string][]corev1.Pod{
			"": {
				makePod("pod-1", "default", "stag-node-1", corev1.PodRunning, "", ""),
				makePod("pod-2", "default", "stag-node-2", corev1.PodPending, "", ""),
			},
		},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	factory := contextFactory(map[string]K8sClient{
		"prod":    prodMock,
		"staging": stagMock,
	})

	c := newWithFactory(Config{
		Contexts: []string{"prod", "staging"},
	}, factory)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	if len(status.Clusters) != 2 {
		t.Fatalf("len(Clusters) = %d, want 2", len(status.Clusters))
	}

	// Both clusters should be connected.
	for i, cluster := range status.Clusters {
		if !cluster.Connected {
			t.Errorf("Clusters[%d] (%s) should be connected", i, cluster.Context)
		}
	}

	// prod: 1 node, 1 pod.
	prod := status.Clusters[0]
	if prod.Context != "prod" {
		t.Errorf("Clusters[0].Context = %q, want %q", prod.Context, "prod")
	}
	if len(prod.Nodes) != 1 {
		t.Errorf("prod nodes = %d, want 1", len(prod.Nodes))
	}
	if prod.TotalPods != 1 {
		t.Errorf("prod TotalPods = %d, want 1", prod.TotalPods)
	}

	// staging: 2 nodes, 2 pods (1 running, 1 pending).
	stag := status.Clusters[1]
	if stag.Context != "staging" {
		t.Errorf("Clusters[1].Context = %q, want %q", stag.Context, "staging")
	}
	if len(stag.Nodes) != 2 {
		t.Errorf("staging nodes = %d, want 2", len(stag.Nodes))
	}
	if stag.TotalPods != 2 {
		t.Errorf("staging TotalPods = %d, want 2", stag.TotalPods)
	}
	if stag.RunningPods != 1 {
		t.Errorf("staging RunningPods = %d, want 1", stag.RunningPods)
	}
	if stag.PendingPods != 1 {
		t.Errorf("staging PendingPods = %d, want 1", stag.PendingPods)
	}

	if !c.Healthy() {
		t.Error("collector should be healthy when all contexts connect")
	}
}

func TestCollect_MultipleContexts_OneDisconnected(t *testing.T) {
	goodMock := &mockClient{
		nodes: []corev1.Node{
			makeNode("node-1", true, nil, "4", "8Gi"),
		},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	factory := contextFactory(map[string]K8sClient{
		"good": goodMock,
		// "bad" is not in the map, so factory returns error.
	})

	c := newWithFactory(Config{
		Contexts: []string{"good", "bad"},
	}, factory)

	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	if len(status.Clusters) != 2 {
		t.Fatalf("len(Clusters) = %d, want 2", len(status.Clusters))
	}

	good := status.Clusters[0]
	if !good.Connected {
		t.Error("good context should be connected")
	}

	bad := status.Clusters[1]
	if bad.Connected {
		t.Error("bad context should NOT be connected")
	}
	if bad.Error == "" {
		t.Error("bad context should have an error message")
	}

	// Still healthy because at least one context connected.
	if !c.Healthy() {
		t.Error("collector should be healthy when at least one context connects")
	}
}

func TestCollect_NodeRoles(t *testing.T) {
	mock := &mockClient{
		nodes: []corev1.Node{
			makeNode("master-1", true, map[string]string{
				"node-role.kubernetes.io/control-plane": "",
				"node-role.kubernetes.io/master":        "",
			}, "4", "8Gi"),
			makeNode("worker-1", true, map[string]string{
				"kubernetes.io/role": "worker",
			}, "8", "16Gi"),
			makeNode("dual-1", true, map[string]string{
				"node-role.kubernetes.io/worker": "",
				"kubernetes.io/role":             "worker", // duplicate role
			}, "4", "8Gi"),
		},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, _ := c.Collect(context.Background())
	status := result.(*ClusterStatus)
	nodes := status.Clusters[0].Nodes

	// master-1: should have both control-plane and master roles.
	if len(nodes[0].Roles) != 2 {
		t.Errorf("master-1 roles = %v, want 2 roles", nodes[0].Roles)
	}

	// worker-1: should have worker role from kubernetes.io/role label.
	if len(nodes[1].Roles) != 1 || nodes[1].Roles[0] != "worker" {
		t.Errorf("worker-1 roles = %v, want [worker]", nodes[1].Roles)
	}

	// dual-1: "worker" appears in both labels but should not be duplicated.
	workerCount := 0
	for _, r := range nodes[2].Roles {
		if r == "worker" {
			workerCount++
		}
	}
	if workerCount != 1 {
		t.Errorf("dual-1 has %d 'worker' entries, want 1 (deduped). roles = %v", workerCount, nodes[2].Roles)
	}
}

func TestCollect_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	mock := &mockClient{
		nodesErr: ctx.Err(),
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(ctx)
	if err != nil {
		t.Fatalf("Collect() should not return Go error even with cancelled context, got: %v", err)
	}

	status := result.(*ClusterStatus)
	cluster := status.Clusters[0]
	if cluster.Connected {
		t.Error("cluster should not be connected with cancelled context")
	}
}

func TestCollect_TimestampSet(t *testing.T) {
	mock := &mockClient{
		nodes:      []corev1.Node{},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{},
	}

	before := time.Now()
	c := newWithFactory(Config{}, mockFactory(mock))
	result, _ := c.Collect(context.Background())
	after := time.Now()

	status := result.(*ClusterStatus)
	if status.Timestamp.Before(before) || status.Timestamp.After(after) {
		t.Errorf("timestamp %v not in range [%v, %v]", status.Timestamp, before, after)
	}
}

func TestCollect_NilNodeLabels(t *testing.T) {
	// Node with nil Labels map should not panic.
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bare-node",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}

	mock := &mockClient{
		nodes:      []corev1.Node{node},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	n := status.Clusters[0].Nodes[0]
	if n.Name != "bare-node" {
		t.Errorf("Name = %q, want %q", n.Name, "bare-node")
	}
	if len(n.Roles) != 0 {
		t.Errorf("Roles = %v, want empty", n.Roles)
	}
}

func TestCollect_NilNodeCapacity(t *testing.T) {
	// Node with nil Capacity should produce empty capacity strings.
	node := corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "no-capacity",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
			// Capacity is nil.
		},
	}

	mock := &mockClient{
		nodes:      []corev1.Node{node},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{makeNamespace("default")},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	n := status.Clusters[0].Nodes[0]
	if n.CPUCapacity != "" {
		t.Errorf("CPUCapacity = %q, want empty", n.CPUCapacity)
	}
	if n.MemCapacity != "" {
		t.Errorf("MemCapacity = %q, want empty", n.MemCapacity)
	}
}

func TestCollect_DeploymentNilReplicas(t *testing.T) {
	// A deployment with nil Replicas field (defaults to 1 in K8s, but
	// the spec pointer can be nil in the API response).
	dep := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "no-replicas",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
		},
		Status: appsv1.DeploymentStatus{
			ReadyReplicas:     1,
			AvailableReplicas: 1,
		},
	}

	mock := &mockClient{
		nodes:      []corev1.Node{makeNode("node-1", true, nil, "4", "8Gi")},
		pods:       map[string][]corev1.Pod{"": {}},
		namespaces: []corev1.Namespace{makeNamespace("default")},
		deployments: map[string][]appsv1.Deployment{
			"": {dep},
		},
	}

	c := newWithFactory(Config{}, mockFactory(mock))
	result, err := c.Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}

	status := result.(*ClusterStatus)
	d := status.Clusters[0].Namespaces[0].Deployments[0]
	if d.Replicas != 0 {
		t.Errorf("Replicas = %d, want 0 (nil spec)", d.Replicas)
	}
	if d.ReadyReplicas != 1 {
		t.Errorf("ReadyReplicas = %d, want 1", d.ReadyReplicas)
	}
}
