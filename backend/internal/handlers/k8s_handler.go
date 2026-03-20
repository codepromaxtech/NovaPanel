package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/services"
)

type K8sHandler struct {
	svc *services.K8sService
}

func NewK8sHandler(svc *services.K8sService) *K8sHandler {
	return &K8sHandler{svc: svc}
}

// -- Clusters --

func (h *K8sHandler) AddCluster(c *gin.Context) {
	var req struct {
		Name        string `json:"name"`
		Kubeconfig  string `json:"kubeconfig"`
		ContextName string `json:"context_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Kubeconfig == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and kubeconfig required"})
		return
	}
	cluster, err := h.svc.AddCluster(c.Request.Context(), req.Name, req.Kubeconfig, req.ContextName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": cluster})
}

func (h *K8sHandler) ListClusters(c *gin.Context) {
	clusters, err := h.svc.ListClusters(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": clusters})
}

func (h *K8sHandler) RemoveCluster(c *gin.Context) {
	if err := h.svc.RemoveCluster(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cluster removed"})
}

func (h *K8sHandler) ClusterInfo(c *gin.Context) {
	info, err := h.svc.GetClusterInfo(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": info})
}

// -- Namespaces --

func (h *K8sHandler) ListNamespaces(c *gin.Context) {
	ns, err := h.svc.ListNamespaces(c.Request.Context(), c.Param("cluster"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": ns})
}

func (h *K8sHandler) CreateNamespace(c *gin.Context) {
	var req struct{ Name string `json:"name"` }
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}
	if err := h.svc.CreateNamespace(c.Request.Context(), c.Param("cluster"), req.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "namespace created"})
}

func (h *K8sHandler) DeleteNamespace(c *gin.Context) {
	if err := h.svc.DeleteNamespace(c.Request.Context(), c.Param("cluster"), c.Param("ns")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "namespace deleted"})
}

// -- Pods --

func (h *K8sHandler) ListPods(c *gin.Context) {
	pods, err := h.svc.ListPods(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pods})
}

func (h *K8sHandler) GetPod(c *gin.Context) {
	pod, err := h.svc.GetPod(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": pod})
}

func (h *K8sHandler) DeletePod(c *gin.Context) {
	if err := h.svc.DeletePod(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pod deleted"})
}

func (h *K8sHandler) PodLogs(c *gin.Context) {
	tail := int64(100)
	if t, err := strconv.ParseInt(c.DefaultQuery("tail", "100"), 10, 64); err == nil {
		tail = t
	}
	logs, err := h.svc.PodLogs(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"), c.Query("container"), tail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs})
}

// -- Deployments --

func (h *K8sHandler) ListDeployments(c *gin.Context) {
	deps, err := h.svc.ListDeployments(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": deps})
}

func (h *K8sHandler) ScaleDeployment(c *gin.Context) {
	var req struct{ Replicas int32 `json:"replicas"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "replicas required"})
		return
	}
	if err := h.svc.ScaleDeployment(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"), req.Replicas); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deployment scaled"})
}

func (h *K8sHandler) RestartDeployment(c *gin.Context) {
	if err := h.svc.RestartDeployment(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deployment restarted"})
}

func (h *K8sHandler) DeleteDeployment(c *gin.Context) {
	if err := h.svc.DeleteDeployment(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deployment deleted"})
}

// -- StatefulSets --

func (h *K8sHandler) ListStatefulSets(c *gin.Context) {
	items, err := h.svc.ListStatefulSets(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) ScaleStatefulSet(c *gin.Context) {
	var req struct{ Replicas int32 `json:"replicas"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "replicas required"}); return }
	if err := h.svc.ScaleStatefulSet(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"), req.Replicas); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "statefulset scaled"})
}

func (h *K8sHandler) DeleteStatefulSet(c *gin.Context) {
	if err := h.svc.DeleteStatefulSet(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "statefulset deleted"})
}

// -- DaemonSets --

func (h *K8sHandler) ListDaemonSets(c *gin.Context) {
	items, err := h.svc.ListDaemonSets(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteDaemonSet(c *gin.Context) {
	if err := h.svc.DeleteDaemonSet(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "daemonset deleted"})
}

// -- ReplicaSets --

func (h *K8sHandler) ListReplicaSets(c *gin.Context) {
	items, err := h.svc.ListReplicaSets(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteReplicaSet(c *gin.Context) {
	if err := h.svc.DeleteReplicaSet(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "replicaset deleted"})
}

// -- Jobs --

func (h *K8sHandler) ListJobs(c *gin.Context) {
	items, err := h.svc.ListJobs(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteJob(c *gin.Context) {
	if err := h.svc.DeleteJob(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "job deleted"})
}

// -- CronJobs --

func (h *K8sHandler) ListCronJobs(c *gin.Context) {
	items, err := h.svc.ListCronJobs(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) SuspendCronJob(c *gin.Context) {
	var req struct{ Suspend bool `json:"suspend"` }
	if err := c.ShouldBindJSON(&req); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := h.svc.SuspendCronJob(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"), req.Suspend); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cronjob updated"})
}

func (h *K8sHandler) TriggerCronJob(c *gin.Context) {
	if err := h.svc.TriggerCronJob(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cronjob triggered"})
}

func (h *K8sHandler) DeleteCronJob(c *gin.Context) {
	if err := h.svc.DeleteCronJob(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "cronjob deleted"})
}

// -- Services --

func (h *K8sHandler) ListServices(c *gin.Context) {
	items, err := h.svc.ListServices(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteSvc(c *gin.Context) {
	if err := h.svc.DeleteService(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service deleted"})
}

// -- Ingresses --

func (h *K8sHandler) ListIngresses(c *gin.Context) {
	items, err := h.svc.ListIngresses(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteIngress(c *gin.Context) {
	if err := h.svc.DeleteIngress(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ingress deleted"})
}

// -- ConfigMaps --

func (h *K8sHandler) ListConfigMaps(c *gin.Context) {
	items, err := h.svc.ListConfigMaps(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) GetConfigMap(c *gin.Context) {
	item, err := h.svc.GetConfigMap(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *K8sHandler) DeleteConfigMap(c *gin.Context) {
	if err := h.svc.DeleteConfigMap(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "configmap deleted"})
}

// -- Secrets --

func (h *K8sHandler) ListSecrets(c *gin.Context) {
	items, err := h.svc.ListSecrets(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) GetSecret(c *gin.Context) {
	item, err := h.svc.GetSecret(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": item})
}

func (h *K8sHandler) DeleteSecret(c *gin.Context) {
	if err := h.svc.DeleteSecret(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "secret deleted"})
}

// -- Storage --

func (h *K8sHandler) ListPVCs(c *gin.Context) {
	items, err := h.svc.ListPVCs(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeletePVC(c *gin.Context) {
	if err := h.svc.DeletePVC(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pvc deleted"})
}

func (h *K8sHandler) ListPVs(c *gin.Context) {
	items, err := h.svc.ListPVs(c.Request.Context(), c.Param("cluster"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeletePV(c *gin.Context) {
	if err := h.svc.DeletePV(c.Request.Context(), c.Param("cluster"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "pv deleted"})
}

func (h *K8sHandler) ListStorageClasses(c *gin.Context) {
	items, err := h.svc.ListStorageClasses(c.Request.Context(), c.Param("cluster"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// -- Nodes --

func (h *K8sHandler) ListNodes(c *gin.Context) {
	items, err := h.svc.ListNodes(c.Request.Context(), c.Param("cluster"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) CordonNode(c *gin.Context) {
	var req struct{ Cordon bool `json:"cordon"` }
	c.ShouldBindJSON(&req)
	if err := h.svc.CordonNode(c.Request.Context(), c.Param("cluster"), c.Param("name"), req.Cordon); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "node updated"})
}

// -- Events --

func (h *K8sHandler) ListEvents(c *gin.Context) {
	items, err := h.svc.ListEvents(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// -- RBAC --

func (h *K8sHandler) ListRoles(c *gin.Context) {
	items, err := h.svc.ListRoles(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) ListClusterRoles(c *gin.Context) {
	items, err := h.svc.ListClusterRoles(c.Request.Context(), c.Param("cluster"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) ListRoleBindings(c *gin.Context) {
	items, err := h.svc.ListRoleBindings(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) ListClusterRoleBindings(c *gin.Context) {
	items, err := h.svc.ListClusterRoleBindings(c.Request.Context(), c.Param("cluster"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) ListServiceAccounts(c *gin.Context) {
	items, err := h.svc.ListServiceAccounts(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteServiceAccount(c *gin.Context) {
	if err := h.svc.DeleteServiceAccount(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service account deleted"})
}

// -- HPA --

func (h *K8sHandler) ListHPAs(c *gin.Context) {
	items, err := h.svc.ListHPAs(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteHPA(c *gin.Context) {
	if err := h.svc.DeleteHPA(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "hpa deleted"})
}

// -- Network Policies --

func (h *K8sHandler) ListNetworkPolicies(c *gin.Context) {
	items, err := h.svc.ListNetworkPolicies(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (h *K8sHandler) DeleteNetworkPolicy(c *gin.Context) {
	if err := h.svc.DeleteNetworkPolicy(c.Request.Context(), c.Param("cluster"), c.Param("ns"), c.Param("name")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"message": "network policy deleted"})
}

// -- Endpoints --

func (h *K8sHandler) ListEndpoints(c *gin.Context) {
	items, err := h.svc.ListEndpoints(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// -- ResourceQuotas --

func (h *K8sHandler) ListResourceQuotas(c *gin.Context) {
	items, err := h.svc.ListResourceQuotas(c.Request.Context(), c.Param("cluster"), c.Query("ns"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// -- CRDs --

func (h *K8sHandler) ListCRDs(c *gin.Context) {
	items, err := h.svc.ListCRDs(c.Request.Context(), c.Param("cluster"))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	c.JSON(http.StatusOK, gin.H{"data": items})
}
