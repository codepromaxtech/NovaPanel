package handlers

import (
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/services"
)

type DockerHandler struct {
	dockerService *services.DockerService
}

func NewDockerHandler(dockerService *services.DockerService) *DockerHandler {
	return &DockerHandler{dockerService: dockerService}
}

// ---- Containers ----

func (h *DockerHandler) ListContainers(c *gin.Context) {
	all := c.Query("all") == "true"
	containers, err := h.dockerService.ListContainers(c.Request.Context(), all)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": containers})
}

func (h *DockerHandler) InspectContainer(c *gin.Context) {
	info, err := h.dockerService.InspectContainer(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "container not found"})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *DockerHandler) StartContainer(c *gin.Context) {
	if err := h.dockerService.StartContainer(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container started"})
}

func (h *DockerHandler) StopContainer(c *gin.Context) {
	if err := h.dockerService.StopContainer(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container stopped"})
}

func (h *DockerHandler) RestartContainer(c *gin.Context) {
	if err := h.dockerService.RestartContainer(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container restarted"})
}

func (h *DockerHandler) RemoveContainer(c *gin.Context) {
	force := c.Query("force") == "true"
	if err := h.dockerService.RemoveContainer(c.Request.Context(), c.Param("id"), force); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container removed"})
}

func (h *DockerHandler) ContainerLogs(c *gin.Context) {
	tail := c.DefaultQuery("tail", "200")
	reader, err := h.dockerService.ContainerLogs(c.Request.Context(), c.Param("id"), tail)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()
	logs, _ := io.ReadAll(reader)
	c.JSON(http.StatusOK, gin.H{"logs": string(logs)})
}

func (h *DockerHandler) CreateContainer(c *gin.Context) {
	var req services.CreateContainerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.dockerService.CreateContainer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "container created and started"})
}

// ---- Images ----

func (h *DockerHandler) ListImages(c *gin.Context) {
	images, err := h.dockerService.ListImages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": images})
}

func (h *DockerHandler) PullImage(c *gin.Context) {
	var req struct {
		Image string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image field required"})
		return
	}
	reader, err := h.dockerService.PullImage(c.Request.Context(), req.Image)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()
	// Read through pull output to complete the pull
	io.Copy(io.Discard, reader)
	c.JSON(http.StatusOK, gin.H{"message": "image pulled: " + req.Image})
}

func (h *DockerHandler) RemoveImage(c *gin.Context) {
	force := c.Query("force") == "true"
	if err := h.dockerService.RemoveImage(c.Request.Context(), c.Param("id"), force); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "image removed"})
}

// ---- Volumes ----

func (h *DockerHandler) ListVolumes(c *gin.Context) {
	vols, err := h.dockerService.ListVolumes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": vols})
}

func (h *DockerHandler) CreateVolume(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Driver string `json:"driver"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Driver == "" {
		req.Driver = "local"
	}
	vol, err := h.dockerService.CreateVolume(c.Request.Context(), req.Name, req.Driver)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, vol)
}

func (h *DockerHandler) RemoveVolume(c *gin.Context) {
	force := c.Query("force") == "true"
	if err := h.dockerService.RemoveVolume(c.Request.Context(), c.Param("name"), force); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "volume removed"})
}

// ---- Networks ----

func (h *DockerHandler) ListNetworks(c *gin.Context) {
	nets, err := h.dockerService.ListNetworks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": nets})
}

func (h *DockerHandler) CreateNetwork(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Driver string `json:"driver"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Driver == "" {
		req.Driver = "bridge"
	}
	net, err := h.dockerService.CreateNetwork(c.Request.Context(), req.Name, req.Driver)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, net)
}

func (h *DockerHandler) RemoveNetwork(c *gin.Context) {
	if err := h.dockerService.RemoveNetwork(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "network removed"})
}

// ---- Stacks ----

func (h *DockerHandler) ListStacks(c *gin.Context) {
	stacks, err := h.dockerService.ListStacks(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": stacks})
}

func (h *DockerHandler) DeployStack(c *gin.Context) {
	var req services.DeployStackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, _ := c.Get("user_id")
	stack, err := h.dockerService.DeployStack(c.Request.Context(), req, userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, stack)
}

func (h *DockerHandler) RemoveStack(c *gin.Context) {
	name := c.Param("name")
	if err := h.dockerService.RemoveStack(c.Request.Context(), name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stack removed"})
}

// ---- Stats ----

func (h *DockerHandler) Stats(c *gin.Context) {
	stats, err := h.dockerService.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ---- Container Stats (per-container CPU/RAM/Net/IO) ----

func (h *DockerHandler) ContainerStats(c *gin.Context) {
	stats, err := h.dockerService.ContainerStats(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ---- Container Rename ----

func (h *DockerHandler) RenameContainer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.dockerService.RenameContainer(c.Request.Context(), c.Param("id"), req.Name); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container renamed"})
}

// ---- Container Duplicate ----

func (h *DockerHandler) DuplicateContainer(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := h.dockerService.DuplicateContainer(c.Request.Context(), c.Param("id"), req.Name)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "container duplicated"})
}

// ---- Container Processes ----

func (h *DockerHandler) ContainerProcesses(c *gin.Context) {
	procs, err := h.dockerService.ContainerProcesses(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, procs)
}

// ---- Image History ----

func (h *DockerHandler) ImageHistory(c *gin.Context) {
	history, err := h.dockerService.ImageHistory(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": history})
}

// ---- Image Tag ----

func (h *DockerHandler) TagImage(c *gin.Context) {
	var req struct {
		Target string `json:"target" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.dockerService.TagImage(c.Request.Context(), c.Param("id"), req.Target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "image tagged"})
}

// ---- Network Connect/Disconnect ----

func (h *DockerHandler) NetworkConnect(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.dockerService.NetworkConnect(c.Request.Context(), c.Param("id"), req.ContainerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container connected to network"})
}

func (h *DockerHandler) NetworkDisconnect(c *gin.Context) {
	var req struct {
		ContainerID string `json:"container_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.dockerService.NetworkDisconnect(c.Request.Context(), c.Param("id"), req.ContainerID, false); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "container disconnected from network"})
}

// ---- System Info ----

func (h *DockerHandler) SystemInfo(c *gin.Context) {
	info, err := h.dockerService.SystemInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// ---- Prune ----

func (h *DockerHandler) PruneSystem(c *gin.Context) {
	result, err := h.dockerService.PruneSystem(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ---- Templates ----

func (h *DockerHandler) ListTemplates(c *gin.Context) {
	templates := services.GetAppTemplates()
	c.JSON(http.StatusOK, gin.H{"data": templates})
}

func (h *DockerHandler) DeployTemplate(c *gin.Context) {
	templateID := c.Param("id")
	var envOverrides struct {
		Env  map[string]string `json:"env"`
		Name string            `json:"name"`
	}
	c.ShouldBindJSON(&envOverrides)

	templates := services.GetAppTemplates()
	var tmpl *services.AppTemplate
	for _, t := range templates {
		if t.ID == templateID {
			tmpl = &t
			break
		}
	}
	if tmpl == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template not found"})
		return
	}

	// Build env list
	envList := make([]string, 0)
	for _, e := range tmpl.Env {
		val := e.Default
		if override, ok := envOverrides.Env[e.Name]; ok {
			val = override
		}
		envList = append(envList, e.Name+"="+val)
	}

	containerName := envOverrides.Name
	if containerName == "" {
		containerName = tmpl.ID
	}

	// Build command
	var cmd []string
	if tmpl.Command != "" {
		cmd = []string{"sh", "-c", tmpl.Command}
	}

	req := services.CreateContainerRequest{
		Name:    containerName,
		Image:   tmpl.Image,
		Env:     envList,
		Ports:   tmpl.Ports,
		Volumes: tmpl.Volumes,
		Restart: tmpl.Restart,
		Cmd:     cmd,
	}

	id, err := h.dockerService.CreateContainer(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "message": "template deployed: " + tmpl.Title})
}

// ---- Pause / Unpause / Kill ----

func (h *DockerHandler) PauseContainer(c *gin.Context) {
	if err := h.dockerService.PauseContainer(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "paused"})
}

func (h *DockerHandler) UnpauseContainer(c *gin.Context) {
	if err := h.dockerService.UnpauseContainer(c.Request.Context(), c.Param("id")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unpaused"})
}

func (h *DockerHandler) KillContainer(c *gin.Context) {
	signal := c.DefaultQuery("signal", "SIGKILL")
	if err := h.dockerService.KillContainer(c.Request.Context(), c.Param("id"), signal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "killed"})
}

// ---- Commit Container ----

func (h *DockerHandler) CommitContainer(c *gin.Context) {
	var req struct {
		Repo    string `json:"repo"`
		Tag     string `json:"tag"`
		Comment string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "repo and tag required"})
		return
	}
	if req.Tag == "" {
		req.Tag = "latest"
	}
	result, err := h.dockerService.CommitContainer(c.Request.Context(), c.Param("id"), req.Repo, req.Tag, req.Comment)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, result)
}

// ---- Container File Browser ----

func (h *DockerHandler) BrowseFiles(c *gin.Context) {
	dirPath := c.DefaultQuery("path", "/")
	files, err := h.dockerService.BrowseContainerFiles(c.Request.Context(), c.Param("id"), dirPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": files, "path": dirPath})
}

// ---- Image Build ----

func (h *DockerHandler) BuildImage(c *gin.Context) {
	var req struct {
		Dockerfile string `json:"dockerfile"`
		Tag        string `json:"tag"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Dockerfile == "" || req.Tag == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "dockerfile and tag required"})
		return
	}
	result, err := h.dockerService.BuildImage(c.Request.Context(), req.Dockerfile, req.Tag)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// ---- Push Image ----

func (h *DockerHandler) PushImage(c *gin.Context) {
	var req struct {
		Ref string `json:"ref"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Ref == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image ref required"})
		return
	}
	output, err := h.dockerService.PushImage(c.Request.Context(), req.Ref)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// ---- Events ----

func (h *DockerHandler) Events(c *gin.Context) {
	sinceStr := c.DefaultQuery("since", "1h")
	dur, err := time.ParseDuration(sinceStr)
	if err != nil {
		dur = time.Hour
	}
	events, err := h.dockerService.StreamEvents(c.Request.Context(), dur)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": events})
}
