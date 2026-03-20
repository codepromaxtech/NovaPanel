package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sCluster represents a registered cluster
type K8sCluster struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kubeconfig  string `json:"kubeconfig,omitempty"`
	ContextName string `json:"context_name"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
}

// K8sService manages Kubernetes clusters
type K8sService struct {
	db *pgxpool.Pool
}

func NewK8sService(db *pgxpool.Pool) *K8sService {
	return &K8sService{db: db}
}

func (s *K8sService) clientFor(ctx context.Context, clusterID string) (*kubernetes.Clientset, error) {
	var kubeconfig string
	err := s.db.QueryRow(ctx, "SELECT kubeconfig FROM k8s_clusters WHERE id=$1", clusterID).Scan(&kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("cluster not found: %w", err)
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

// ---- Cluster CRUD ----

func (s *K8sService) AddCluster(ctx context.Context, name, kubeconfig, contextName string) (*K8sCluster, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	_, err = cs.Discovery().ServerVersion()
	status := "active"
	if err != nil {
		status = "unreachable"
	}
	c := &K8sCluster{}
	err = s.db.QueryRow(ctx,
		"INSERT INTO k8s_clusters(name,kubeconfig,context_name,status) VALUES($1,$2,$3,$4) RETURNING id,name,context_name,status,created_at",
		name, kubeconfig, contextName, status).Scan(&c.ID, &c.Name, &c.ContextName, &c.Status, &c.CreatedAt)
	return c, err
}

func (s *K8sService) ListClusters(ctx context.Context) ([]K8sCluster, error) {
	rows, err := s.db.Query(ctx, "SELECT id,name,context_name,status,created_at FROM k8s_clusters ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []K8sCluster
	for rows.Next() {
		var c K8sCluster
		rows.Scan(&c.ID, &c.Name, &c.ContextName, &c.Status, &c.CreatedAt)
		out = append(out, c)
	}
	return out, nil
}

func (s *K8sService) RemoveCluster(ctx context.Context, id string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM k8s_clusters WHERE id=$1", id)
	return err
}

type ClusterInfo struct {
	Version    string `json:"version"`
	Platform   string `json:"platform"`
	NodeCount  int    `json:"node_count"`
	TotalCPU   string `json:"total_cpu"`
	TotalRAM   string `json:"total_ram"`
	PodCount   int    `json:"pod_count"`
}

func (s *K8sService) GetClusterInfo(ctx context.Context, clusterID string) (*ClusterInfo, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	ver, _ := cs.Discovery().ServerVersion()
	nodes, _ := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	pods, _ := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	var cpuTotal, ramTotal int64
	for _, n := range nodes.Items {
		cpuTotal += n.Status.Capacity.Cpu().MilliValue()
		ramTotal += n.Status.Capacity.Memory().Value()
	}
	return &ClusterInfo{
		Version:   ver.GitVersion,
		Platform:  ver.Platform,
		NodeCount: len(nodes.Items),
		TotalCPU:  fmt.Sprintf("%.1f cores", float64(cpuTotal)/1000),
		TotalRAM:  fmt.Sprintf("%.1f GB", float64(ramTotal)/1073741824),
		PodCount:  len(pods.Items),
	}, nil
}

// ---- Namespaces ----

func (s *K8sService) ListNamespaces(ctx context.Context, clusterID string) ([]string, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []string
	for _, n := range list.Items { out = append(out, n.Name) }
	return out, nil
}

func (s *K8sService) CreateNamespace(ctx context.Context, clusterID, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	_, err = cs.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}, metav1.CreateOptions{})
	return err
}

func (s *K8sService) DeleteNamespace(ctx context.Context, clusterID, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Pods ----

type PodSummary struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Status     string `json:"status"`
	Node       string `json:"node"`
	IP         string `json:"ip"`
	Restarts   int32  `json:"restarts"`
	Containers int    `json:"containers"`
	Ready      string `json:"ready"`
	Age        string `json:"age"`
}

func (s *K8sService) ListPods(ctx context.Context, clusterID, ns string) ([]PodSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []PodSummary
	for _, p := range list.Items {
		var restarts int32
		readyCount := 0
		for _, c := range p.Status.ContainerStatuses {
			restarts += c.RestartCount
			if c.Ready { readyCount++ }
		}
		out = append(out, PodSummary{
			Name: p.Name, Namespace: p.Namespace, Status: string(p.Status.Phase),
			Node: p.Spec.NodeName, IP: p.Status.PodIP, Restarts: restarts,
			Containers: len(p.Spec.Containers),
			Ready: fmt.Sprintf("%d/%d", readyCount, len(p.Spec.Containers)),
			Age: time.Since(p.CreationTimestamp.Time).Truncate(time.Second).String(),
		})
	}
	return out, nil
}

func (s *K8sService) GetPod(ctx context.Context, clusterID, ns, name string) (*corev1.Pod, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	return cs.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{})
}

func (s *K8sService) DeletePod(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

func (s *K8sService) PodLogs(ctx context.Context, clusterID, ns, name, container string, tail int64) (string, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return "", err }
	opts := &corev1.PodLogOptions{TailLines: &tail}
	if container != "" { opts.Container = container }
	req := cs.CoreV1().Pods(ns).GetLogs(name, opts)
	stream, err := req.Stream(ctx)
	if err != nil { return "", err }
	defer stream.Close()
	buf := make([]byte, 64*1024)
	n, _ := stream.Read(buf)
	return string(buf[:n]), nil
}

// ---- Deployments ----

type DeploymentSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
	Ready     int32  `json:"ready"`
	Available int32  `json:"available"`
	Image     string `json:"image"`
	Age       string `json:"age"`
}

func (s *K8sService) ListDeployments(ctx context.Context, clusterID, ns string) ([]DeploymentSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []DeploymentSummary
	for _, d := range list.Items {
		img := ""
		if len(d.Spec.Template.Spec.Containers) > 0 { img = d.Spec.Template.Spec.Containers[0].Image }
		out = append(out, DeploymentSummary{
			Name: d.Name, Namespace: d.Namespace, Replicas: *d.Spec.Replicas,
			Ready: d.Status.ReadyReplicas, Available: d.Status.AvailableReplicas,
			Image: img, Age: time.Since(d.CreationTimestamp.Time).Truncate(time.Second).String(),
		})
	}
	return out, nil
}

func (s *K8sService) ScaleDeployment(ctx context.Context, clusterID, ns, name string, replicas int32) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	scale, err := cs.AppsV1().Deployments(ns).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil { return err }
	scale.Spec.Replicas = replicas
	_, err = cs.AppsV1().Deployments(ns).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func (s *K8sService) RestartDeployment(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	dep, err := cs.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil { return err }
	if dep.Spec.Template.Annotations == nil { dep.Spec.Template.Annotations = map[string]string{} }
	dep.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	_, err = cs.AppsV1().Deployments(ns).Update(ctx, dep, metav1.UpdateOptions{})
	return err
}

func (s *K8sService) DeleteDeployment(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.AppsV1().Deployments(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- StatefulSets ----

type WorkloadSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Replicas  int32  `json:"replicas"`
	Ready     int32  `json:"ready"`
	Age       string `json:"age"`
}

func (s *K8sService) ListStatefulSets(ctx context.Context, clusterID, ns string) ([]WorkloadSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.AppsV1().StatefulSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []WorkloadSummary
	for _, s := range list.Items {
		out = append(out, WorkloadSummary{Name: s.Name, Namespace: s.Namespace, Replicas: *s.Spec.Replicas, Ready: s.Status.ReadyReplicas, Age: time.Since(s.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) ScaleStatefulSet(ctx context.Context, clusterID, ns, name string, replicas int32) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	sc, err := cs.AppsV1().StatefulSets(ns).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil { return err }
	sc.Spec.Replicas = replicas
	_, err = cs.AppsV1().StatefulSets(ns).UpdateScale(ctx, name, sc, metav1.UpdateOptions{})
	return err
}

func (s *K8sService) DeleteStatefulSet(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.AppsV1().StatefulSets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- DaemonSets ----

func (s *K8sService) ListDaemonSets(ctx context.Context, clusterID, ns string) ([]WorkloadSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []WorkloadSummary
	for _, d := range list.Items {
		out = append(out, WorkloadSummary{Name: d.Name, Namespace: d.Namespace, Replicas: d.Status.DesiredNumberScheduled, Ready: d.Status.NumberReady, Age: time.Since(d.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteDaemonSet(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.AppsV1().DaemonSets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- ReplicaSets ----

func (s *K8sService) ListReplicaSets(ctx context.Context, clusterID, ns string) ([]WorkloadSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []WorkloadSummary
	for _, r := range list.Items {
		reps := int32(0)
		if r.Spec.Replicas != nil { reps = *r.Spec.Replicas }
		out = append(out, WorkloadSummary{Name: r.Name, Namespace: r.Namespace, Replicas: reps, Ready: r.Status.ReadyReplicas, Age: time.Since(r.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteReplicaSet(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.AppsV1().ReplicaSets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Jobs ----

type JobSummary struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Succeeded  int32  `json:"succeeded"`
	Failed     int32  `json:"failed"`
	Active     int32  `json:"active"`
	Completions string `json:"completions"`
	Age        string `json:"age"`
}

func (s *K8sService) ListJobs(ctx context.Context, clusterID, ns string) ([]JobSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.BatchV1().Jobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []JobSummary
	for _, j := range list.Items {
		comp := "1"
		if j.Spec.Completions != nil { comp = fmt.Sprintf("%d", *j.Spec.Completions) }
		out = append(out, JobSummary{Name: j.Name, Namespace: j.Namespace, Succeeded: j.Status.Succeeded, Failed: j.Status.Failed, Active: j.Status.Active, Completions: fmt.Sprintf("%d/%s", j.Status.Succeeded, comp), Age: time.Since(j.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteJob(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	bg := metav1.DeletePropagationBackground
	return cs.BatchV1().Jobs(ns).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &bg})
}

// ---- CronJobs ----

type CronJobSummary struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Schedule     string `json:"schedule"`
	Suspended    bool   `json:"suspended"`
	Active       int    `json:"active"`
	LastSchedule string `json:"last_schedule"`
	Age          string `json:"age"`
}

func (s *K8sService) ListCronJobs(ctx context.Context, clusterID, ns string) ([]CronJobSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.BatchV1().CronJobs(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []CronJobSummary
	for _, c := range list.Items {
		ls := "Never"
		if c.Status.LastScheduleTime != nil { ls = c.Status.LastScheduleTime.Format(time.RFC3339) }
		out = append(out, CronJobSummary{Name: c.Name, Namespace: c.Namespace, Schedule: c.Spec.Schedule, Suspended: c.Spec.Suspend != nil && *c.Spec.Suspend, Active: len(c.Status.Active), LastSchedule: ls, Age: time.Since(c.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) SuspendCronJob(ctx context.Context, clusterID, ns, name string, suspend bool) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	cj, err := cs.BatchV1().CronJobs(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil { return err }
	cj.Spec.Suspend = &suspend
	_, err = cs.BatchV1().CronJobs(ns).Update(ctx, cj, metav1.UpdateOptions{})
	return err
}

func (s *K8sService) TriggerCronJob(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	cj, err := cs.BatchV1().CronJobs(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil { return err }
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: name + "-manual-" + fmt.Sprintf("%d", time.Now().Unix()), Namespace: ns},
		Spec:       cj.Spec.JobTemplate.Spec,
	}
	_, err = cs.BatchV1().Jobs(ns).Create(ctx, job, metav1.CreateOptions{})
	return err
}

func (s *K8sService) DeleteCronJob(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.BatchV1().CronJobs(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Services ----

type ServiceSummary struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Type       string `json:"type"`
	ClusterIP  string `json:"cluster_ip"`
	ExternalIP string `json:"external_ip"`
	Ports      string `json:"ports"`
	Age        string `json:"age"`
}

func (s *K8sService) ListServices(ctx context.Context, clusterID, ns string) ([]ServiceSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []ServiceSummary
	for _, svc := range list.Items {
		var ports []string
		for _, p := range svc.Spec.Ports { ports = append(ports, fmt.Sprintf("%d/%s", p.Port, p.Protocol)) }
		ext := "<none>"
		if len(svc.Status.LoadBalancer.Ingress) > 0 { ext = svc.Status.LoadBalancer.Ingress[0].IP }
		out = append(out, ServiceSummary{Name: svc.Name, Namespace: svc.Namespace, Type: string(svc.Spec.Type), ClusterIP: svc.Spec.ClusterIP, ExternalIP: ext, Ports: strings.Join(ports, ", "), Age: time.Since(svc.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteService(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().Services(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Ingresses ----

type IngressSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Hosts     string `json:"hosts"`
	Address   string `json:"address"`
	Age       string `json:"age"`
}

func (s *K8sService) ListIngresses(ctx context.Context, clusterID, ns string) ([]IngressSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []IngressSummary
	for _, ing := range list.Items {
		var hosts []string
		for _, r := range ing.Spec.Rules { hosts = append(hosts, r.Host) }
		addr := ""
		if len(ing.Status.LoadBalancer.Ingress) > 0 { addr = ing.Status.LoadBalancer.Ingress[0].IP }
		out = append(out, IngressSummary{Name: ing.Name, Namespace: ing.Namespace, Hosts: strings.Join(hosts, ", "), Address: addr, Age: time.Since(ing.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteIngress(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.NetworkingV1().Ingresses(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- ConfigMaps ----

type ConfigMapSummary struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	DataKeys  int               `json:"data_keys"`
	Data      map[string]string `json:"data,omitempty"`
	Age       string            `json:"age"`
}

func (s *K8sService) ListConfigMaps(ctx context.Context, clusterID, ns string) ([]ConfigMapSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []ConfigMapSummary
	for _, cm := range list.Items {
		out = append(out, ConfigMapSummary{Name: cm.Name, Namespace: cm.Namespace, DataKeys: len(cm.Data), Age: time.Since(cm.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) GetConfigMap(ctx context.Context, clusterID, ns, name string) (*ConfigMapSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	cm, err := cs.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil { return nil, err }
	return &ConfigMapSummary{Name: cm.Name, Namespace: cm.Namespace, DataKeys: len(cm.Data), Data: cm.Data, Age: time.Since(cm.CreationTimestamp.Time).Truncate(time.Second).String()}, nil
}

func (s *K8sService) DeleteConfigMap(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().ConfigMaps(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Secrets ----

type SecretSummary struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Type      string            `json:"type"`
	DataKeys  int               `json:"data_keys"`
	Data      map[string]string `json:"data,omitempty"`
	Age       string            `json:"age"`
}

func (s *K8sService) ListSecrets(ctx context.Context, clusterID, ns string) ([]SecretSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Secrets(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []SecretSummary
	for _, sec := range list.Items {
		out = append(out, SecretSummary{Name: sec.Name, Namespace: sec.Namespace, Type: string(sec.Type), DataKeys: len(sec.Data), Age: time.Since(sec.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) GetSecret(ctx context.Context, clusterID, ns, name string) (*SecretSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	sec, err := cs.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil { return nil, err }
	data := map[string]string{}
	for k, v := range sec.Data { data[k] = string(v) }
	return &SecretSummary{Name: sec.Name, Namespace: sec.Namespace, Type: string(sec.Type), DataKeys: len(sec.Data), Data: data, Age: time.Since(sec.CreationTimestamp.Time).Truncate(time.Second).String()}, nil
}

func (s *K8sService) DeleteSecret(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().Secrets(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- PVCs ----

type PVCSummary struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Volume       string `json:"volume"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"access_modes"`
	StorageClass string `json:"storage_class"`
	Age          string `json:"age"`
}

func (s *K8sService) ListPVCs(ctx context.Context, clusterID, ns string) ([]PVCSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []PVCSummary
	for _, pvc := range list.Items {
		cap := ""
		if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok { cap = q.String() }
		sc := ""
		if pvc.Spec.StorageClassName != nil { sc = *pvc.Spec.StorageClassName }
		var modes []string
		for _, m := range pvc.Spec.AccessModes { modes = append(modes, string(m)) }
		out = append(out, PVCSummary{Name: pvc.Name, Namespace: pvc.Namespace, Status: string(pvc.Status.Phase), Volume: pvc.Spec.VolumeName, Capacity: cap, AccessModes: strings.Join(modes, ","), StorageClass: sc, Age: time.Since(pvc.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeletePVC(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().PersistentVolumeClaims(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- PVs ----

type PVSummary struct {
	Name         string `json:"name"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"access_modes"`
	Status       string `json:"status"`
	Claim        string `json:"claim"`
	StorageClass string `json:"storage_class"`
	Age          string `json:"age"`
}

func (s *K8sService) ListPVs(ctx context.Context, clusterID string) ([]PVSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []PVSummary
	for _, pv := range list.Items {
		cap := pv.Spec.Capacity.Storage().String()
		claim := ""
		if pv.Spec.ClaimRef != nil { claim = pv.Spec.ClaimRef.Namespace + "/" + pv.Spec.ClaimRef.Name }
		var modes []string
		for _, m := range pv.Spec.AccessModes { modes = append(modes, string(m)) }
		out = append(out, PVSummary{Name: pv.Name, Capacity: cap, AccessModes: strings.Join(modes, ","), Status: string(pv.Status.Phase), Claim: claim, StorageClass: pv.Spec.StorageClassName, Age: time.Since(pv.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeletePV(ctx context.Context, clusterID, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().PersistentVolumes().Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- StorageClasses ----

type StorageClassSummary struct {
	Name        string `json:"name"`
	Provisioner string `json:"provisioner"`
	ReclaimPolicy string `json:"reclaim_policy"`
	VolumeBinding string `json:"volume_binding"`
	IsDefault   bool   `json:"is_default"`
}

func (s *K8sService) ListStorageClasses(ctx context.Context, clusterID string) ([]StorageClassSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []StorageClassSummary
	for _, sc := range list.Items {
		rp := "Delete"
		if sc.ReclaimPolicy != nil { rp = string(*sc.ReclaimPolicy) }
		vb := ""
		if sc.VolumeBindingMode != nil { vb = string(*sc.VolumeBindingMode) }
		isDef := sc.Annotations["storageclass.kubernetes.io/is-default-class"] == "true"
		out = append(out, StorageClassSummary{Name: sc.Name, Provisioner: sc.Provisioner, ReclaimPolicy: rp, VolumeBinding: vb, IsDefault: isDef})
	}
	return out, nil
}

// ---- Nodes ----

type NodeSummary struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Roles    string `json:"roles"`
	Version  string `json:"version"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	CPU      string `json:"cpu"`
	Memory   string `json:"memory"`
	PodCount int    `json:"pod_count"`
	Age      string `json:"age"`
}

func (s *K8sService) ListNodes(ctx context.Context, clusterID string) ([]NodeSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	pods, _ := cs.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	nodePods := map[string]int{}
	if pods != nil { for _, p := range pods.Items { nodePods[p.Spec.NodeName]++ } }
	var out []NodeSummary
	for _, n := range list.Items {
		status := "NotReady"
		for _, c := range n.Status.Conditions { if c.Type == corev1.NodeReady && c.Status == corev1.ConditionTrue { status = "Ready" } }
		var roles []string
		for k := range n.Labels { if strings.HasPrefix(k, "node-role.kubernetes.io/") { roles = append(roles, strings.TrimPrefix(k, "node-role.kubernetes.io/")) } }
		if len(roles) == 0 { roles = []string{"<none>"} }
		out = append(out, NodeSummary{Name: n.Name, Status: status, Roles: strings.Join(roles, ","), Version: n.Status.NodeInfo.KubeletVersion, OS: n.Status.NodeInfo.OSImage, Arch: n.Status.NodeInfo.Architecture, CPU: n.Status.Capacity.Cpu().String(), Memory: fmt.Sprintf("%.1f GB", float64(n.Status.Capacity.Memory().Value())/1073741824), PodCount: nodePods[n.Name], Age: time.Since(n.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) CordonNode(ctx context.Context, clusterID, name string, cordon bool) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	node, err := cs.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil { return err }
	node.Spec.Unschedulable = cordon
	_, err = cs.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	return err
}

// ---- Events ----

type EventSummary struct {
	Type      string `json:"type"`
	Reason    string `json:"reason"`
	Object    string `json:"object"`
	Message   string `json:"message"`
	Count     int32  `json:"count"`
	FirstSeen string `json:"first_seen"`
	LastSeen  string `json:"last_seen"`
}

func (s *K8sService) ListEvents(ctx context.Context, clusterID, ns string) ([]EventSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []EventSummary
	for _, e := range list.Items {
		out = append(out, EventSummary{Type: e.Type, Reason: e.Reason, Object: e.InvolvedObject.Kind + "/" + e.InvolvedObject.Name, Message: e.Message, Count: e.Count, FirstSeen: e.FirstTimestamp.Format(time.RFC3339), LastSeen: e.LastTimestamp.Format(time.RFC3339)})
	}
	return out, nil
}

// ---- RBAC ----

type RBACItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Kind      string `json:"kind"`
	Age       string `json:"age"`
}

func (s *K8sService) ListRoles(ctx context.Context, clusterID, ns string) ([]RBACItem, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.RbacV1().Roles(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []RBACItem
	for _, r := range list.Items { out = append(out, RBACItem{Name: r.Name, Namespace: r.Namespace, Kind: "Role", Age: time.Since(r.CreationTimestamp.Time).Truncate(time.Second).String()}) }
	return out, nil
}

func (s *K8sService) ListClusterRoles(ctx context.Context, clusterID string) ([]RBACItem, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []RBACItem
	for _, r := range list.Items { out = append(out, RBACItem{Name: r.Name, Kind: "ClusterRole", Age: time.Since(r.CreationTimestamp.Time).Truncate(time.Second).String()}) }
	return out, nil
}

func (s *K8sService) ListRoleBindings(ctx context.Context, clusterID, ns string) ([]RBACItem, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.RbacV1().RoleBindings(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []RBACItem
	for _, r := range list.Items { out = append(out, RBACItem{Name: r.Name, Namespace: r.Namespace, Kind: "RoleBinding", Age: time.Since(r.CreationTimestamp.Time).Truncate(time.Second).String()}) }
	return out, nil
}

func (s *K8sService) ListClusterRoleBindings(ctx context.Context, clusterID string) ([]RBACItem, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []RBACItem
	for _, r := range list.Items { out = append(out, RBACItem{Name: r.Name, Kind: "ClusterRoleBinding", Age: time.Since(r.CreationTimestamp.Time).Truncate(time.Second).String()}) }
	return out, nil
}

func (s *K8sService) ListServiceAccounts(ctx context.Context, clusterID, ns string) ([]RBACItem, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().ServiceAccounts(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []RBACItem
	for _, sa := range list.Items { out = append(out, RBACItem{Name: sa.Name, Namespace: sa.Namespace, Kind: "ServiceAccount", Age: time.Since(sa.CreationTimestamp.Time).Truncate(time.Second).String()}) }
	return out, nil
}

func (s *K8sService) DeleteServiceAccount(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.CoreV1().ServiceAccounts(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- HPA ----

type HPASummary struct {
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	Reference  string `json:"reference"`
	MinReplicas int32 `json:"min_replicas"`
	MaxReplicas int32 `json:"max_replicas"`
	Current    int32  `json:"current_replicas"`
	Age        string `json:"age"`
}

func (s *K8sService) ListHPAs(ctx context.Context, clusterID, ns string) ([]HPASummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []HPASummary
	for _, h := range list.Items {
		minR := int32(1)
		if h.Spec.MinReplicas != nil { minR = *h.Spec.MinReplicas }
		out = append(out, HPASummary{Name: h.Name, Namespace: h.Namespace, Reference: h.Spec.ScaleTargetRef.Kind + "/" + h.Spec.ScaleTargetRef.Name, MinReplicas: minR, MaxReplicas: h.Spec.MaxReplicas, Current: h.Status.CurrentReplicas, Age: time.Since(h.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteHPA(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.AutoscalingV2().HorizontalPodAutoscalers(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Network Policies ----

type NetworkPolicySummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	PodSelector string `json:"pod_selector"`
	Age       string `json:"age"`
}

func (s *K8sService) ListNetworkPolicies(ctx context.Context, clusterID, ns string) ([]NetworkPolicySummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.NetworkingV1().NetworkPolicies(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []NetworkPolicySummary
	for _, np := range list.Items {
		sel := "<all>"
		if len(np.Spec.PodSelector.MatchLabels) > 0 {
			var pairs []string
			for k, v := range np.Spec.PodSelector.MatchLabels { pairs = append(pairs, k+"="+v) }
			sel = strings.Join(pairs, ",")
		}
		out = append(out, NetworkPolicySummary{Name: np.Name, Namespace: np.Namespace, PodSelector: sel, Age: time.Since(np.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

func (s *K8sService) DeleteNetworkPolicy(ctx context.Context, clusterID, ns, name string) error {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return err }
	return cs.NetworkingV1().NetworkPolicies(ns).Delete(ctx, name, metav1.DeleteOptions{})
}

// ---- Endpoints ----

type EndpointSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Addresses int    `json:"addresses"`
	Age       string `json:"age"`
}

func (s *K8sService) ListEndpoints(ctx context.Context, clusterID, ns string) ([]EndpointSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().Endpoints(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []EndpointSummary
	for _, ep := range list.Items {
		count := 0
		for _, sub := range ep.Subsets { count += len(sub.Addresses) }
		out = append(out, EndpointSummary{Name: ep.Name, Namespace: ep.Namespace, Addresses: count, Age: time.Since(ep.CreationTimestamp.Time).Truncate(time.Second).String()})
	}
	return out, nil
}

// ---- ResourceQuotas / LimitRanges ----

type ResourceQuotaSummary struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Hard      map[string]string `json:"hard"`
	Used      map[string]string `json:"used"`
}

func (s *K8sService) ListResourceQuotas(ctx context.Context, clusterID, ns string) ([]ResourceQuotaSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	list, err := cs.CoreV1().ResourceQuotas(ns).List(ctx, metav1.ListOptions{})
	if err != nil { return nil, err }
	var out []ResourceQuotaSummary
	for _, rq := range list.Items {
		hard := map[string]string{}
		used := map[string]string{}
		for k, v := range rq.Status.Hard { hard[string(k)] = v.String() }
		for k, v := range rq.Status.Used { used[string(k)] = v.String() }
		out = append(out, ResourceQuotaSummary{Name: rq.Name, Namespace: rq.Namespace, Hard: hard, Used: used})
	}
	return out, nil
}

// ---- CRDs ----

type CRDSummary struct {
	Name    string `json:"name"`
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	Scope   string `json:"scope"`
}

func (s *K8sService) ListCRDs(ctx context.Context, clusterID string) ([]CRDSummary, error) {
	cs, err := s.clientFor(ctx, clusterID)
	if err != nil { return nil, err }
	// Use discovery to list API resources including CRDs
	_, apiLists, err := cs.Discovery().ServerGroupsAndResources()
	if err != nil { return nil, err }
	var out []CRDSummary
	for _, list := range apiLists {
		gv := list.GroupVersion
		parts := strings.SplitN(gv, "/", 2)
		group := ""
		ver := gv
		if len(parts) == 2 { group = parts[0]; ver = parts[1] }
		for _, r := range list.APIResources {
			if strings.Contains(r.Name, "/") { continue } // skip subresources
			if group == "" || group == "apps" || group == "batch" || group == "autoscaling" || group == "networking.k8s.io" || group == "rbac.authorization.k8s.io" || group == "storage.k8s.io" || group == "policy" { continue } // skip built-in
			scope := "Namespaced"
			if !r.Namespaced { scope = "Cluster" }
			out = append(out, CRDSummary{Name: r.Name, Group: group, Version: ver, Kind: r.Kind, Scope: scope})
		}
	}
	return out, nil
}

// Suppress unused import warnings
var _ = appsv1.SchemeGroupVersion
var _ = autoscalingv2.SchemeGroupVersion
var _ = batchv1.SchemeGroupVersion
var _ = corev1.SchemeGroupVersion
var _ = networkingv1.SchemeGroupVersion
var _ = rbacv1.SchemeGroupVersion
var _ = storagev1.SchemeGroupVersion
var _ = resource.DecimalSI
