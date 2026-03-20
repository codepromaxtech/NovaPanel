package services

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DockerService manages Docker resources via the Engine SDK.
type DockerService struct {
	cli *client.Client
	db  *pgxpool.Pool
}

// NewDockerService creates a new DockerService connected to the local Docker socket.
func NewDockerService(db *pgxpool.Pool) (*DockerService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	return &DockerService{cli: cli, db: db}, nil
}

// ---- Containers ----

// ContainerSummary holds the data we send to the frontend.
type ContainerSummary struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	State   string            `json:"state"`
	Status  string            `json:"status"`
	Ports   []PortBinding     `json:"ports"`
	Created int64             `json:"created"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type PortBinding struct {
	Private uint16 `json:"private"`
	Public  uint16 `json:"public"`
	Type    string `json:"type"`
}

func (d *DockerService) ListContainers(ctx context.Context, all bool) ([]ContainerSummary, error) {
	containers, err := d.cli.ContainerList(ctx, container.ListOptions{All: all})
	if err != nil {
		return nil, err
	}
	out := make([]ContainerSummary, 0, len(containers))
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}
		ports := make([]PortBinding, 0, len(c.Ports))
		for _, p := range c.Ports {
			ports = append(ports, PortBinding{
				Private: p.PrivatePort,
				Public:  p.PublicPort,
				Type:    p.Type,
			})
		}
		out = append(out, ContainerSummary{
			ID:      c.ID[:12],
			Name:    name,
			Image:   c.Image,
			State:   c.State,
			Status:  c.Status,
			Ports:   ports,
			Created: c.Created,
			Labels:  c.Labels,
		})
	}
	return out, nil
}

func (d *DockerService) InspectContainer(ctx context.Context, id string) (container.InspectResponse, error) {
	return d.cli.ContainerInspect(ctx, id)
}

func (d *DockerService) StartContainer(ctx context.Context, id string) error {
	return d.cli.ContainerStart(ctx, id, container.StartOptions{})
}

func (d *DockerService) StopContainer(ctx context.Context, id string) error {
	return d.cli.ContainerStop(ctx, id, container.StopOptions{})
}

func (d *DockerService) RestartContainer(ctx context.Context, id string) error {
	return d.cli.ContainerRestart(ctx, id, container.StopOptions{})
}

func (d *DockerService) RemoveContainer(ctx context.Context, id string, force bool) error {
	return d.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: force})
}

func (d *DockerService) ContainerLogs(ctx context.Context, id string, tail string) (io.ReadCloser, error) {
	if tail == "" {
		tail = "200"
	}
	return d.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       tail,
		Timestamps: true,
	})
}

type CreateContainerRequest struct {
	Name       string            `json:"name"`
	Image      string            `json:"image"`
	Cmd        []string          `json:"cmd,omitempty"`
	Env        []string          `json:"env,omitempty"`
	Ports      map[string]string `json:"ports,omitempty"`   // "80/tcp" -> "8080"
	Volumes    []string          `json:"volumes,omitempty"` // "/host:/container"
	Restart    string            `json:"restart,omitempty"` // "always", "unless-stopped"
	Network    string            `json:"network,omitempty"`
	MemoryMB   int64             `json:"memory_mb,omitempty"`
	CPUs       float64           `json:"cpus,omitempty"`
	Hostname   string            `json:"hostname,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

func (d *DockerService) CreateContainer(ctx context.Context, req CreateContainerRequest) (string, error) {
	config := &container.Config{
		Image:    req.Image,
		Env:      req.Env,
		Cmd:      req.Cmd,
		Hostname: req.Hostname,
		Labels:   req.Labels,
	}

	hostConfig := &container.HostConfig{}

	// Resource limits
	if req.MemoryMB > 0 {
		hostConfig.Resources.Memory = req.MemoryMB * 1024 * 1024
	}
	if req.CPUs > 0 {
		hostConfig.Resources.NanoCPUs = int64(req.CPUs * 1e9)
	}

	// Restart policy
	switch req.Restart {
	case "always":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyAlways}
	case "unless-stopped":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyUnlessStopped}
	case "on-failure":
		hostConfig.RestartPolicy = container.RestartPolicy{Name: container.RestartPolicyOnFailure}
	}

	// Volumes / Binds
	if len(req.Volumes) > 0 {
		hostConfig.Binds = req.Volumes
	}

	// Network
	var netConfig *network.NetworkingConfig
	if req.Network != "" {
		netConfig = &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				req.Network: {},
			},
		}
	}

	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, netConfig, nil, req.Name)
	if err != nil {
		return "", err
	}
	// Auto-start
	if startErr := d.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); startErr != nil {
		return resp.ID, startErr
	}
	return resp.ID, nil
}

// ---- Images ----

type ImageSummary struct {
	ID      string   `json:"id"`
	Tags    []string `json:"tags"`
	Size    int64    `json:"size"`
	Created int64    `json:"created"`
}

func (d *DockerService) ListImages(ctx context.Context) ([]ImageSummary, error) {
	images, err := d.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]ImageSummary, 0, len(images))
	for _, img := range images {
		id := img.ID
		if strings.HasPrefix(id, "sha256:") {
			id = id[7:19] // short hash
		}
		out = append(out, ImageSummary{
			ID:      id,
			Tags:    img.RepoTags,
			Size:    img.Size,
			Created: img.Created,
		})
	}
	return out, nil
}

func (d *DockerService) PullImage(ctx context.Context, ref string) (io.ReadCloser, error) {
	return d.cli.ImagePull(ctx, ref, image.PullOptions{})
}

func (d *DockerService) RemoveImage(ctx context.Context, id string, force bool) error {
	_, err := d.cli.ImageRemove(ctx, id, image.RemoveOptions{Force: force})
	return err
}

// ---- Volumes ----

type VolumeSummary struct {
	Name       string `json:"name"`
	Driver     string `json:"driver"`
	Mountpoint string `json:"mountpoint"`
	CreatedAt  string `json:"created_at"`
}

func (d *DockerService) ListVolumes(ctx context.Context) ([]VolumeSummary, error) {
	resp, err := d.cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]VolumeSummary, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		out = append(out, VolumeSummary{
			Name:       v.Name,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			CreatedAt:  v.CreatedAt,
		})
	}
	return out, nil
}

func (d *DockerService) CreateVolume(ctx context.Context, name, driver string) (*VolumeSummary, error) {
	v, err := d.cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: driver,
	})
	if err != nil {
		return nil, err
	}
	return &VolumeSummary{Name: v.Name, Driver: v.Driver, Mountpoint: v.Mountpoint, CreatedAt: v.CreatedAt}, nil
}

func (d *DockerService) RemoveVolume(ctx context.Context, name string, force bool) error {
	return d.cli.VolumeRemove(ctx, name, force)
}

// ---- Networks ----

type NetworkSummary struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Driver  string `json:"driver"`
	Scope   string `json:"scope"`
	Subnet  string `json:"subnet,omitempty"`
	Gateway string `json:"gateway,omitempty"`
}

func (d *DockerService) ListNetworks(ctx context.Context) ([]NetworkSummary, error) {
	nets, err := d.cli.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, err
	}
	out := make([]NetworkSummary, 0, len(nets))
	for _, n := range nets {
		ns := NetworkSummary{
			ID:     n.ID[:12],
			Name:   n.Name,
			Driver: n.Driver,
			Scope:  n.Scope,
		}
		if len(n.IPAM.Config) > 0 {
			ns.Subnet = n.IPAM.Config[0].Subnet
			ns.Gateway = n.IPAM.Config[0].Gateway
		}
		out = append(out, ns)
	}
	return out, nil
}

func (d *DockerService) CreateNetwork(ctx context.Context, name, driver string) (*NetworkSummary, error) {
	resp, err := d.cli.NetworkCreate(ctx, name, network.CreateOptions{Driver: driver})
	if err != nil {
		return nil, err
	}
	return &NetworkSummary{ID: resp.ID[:12], Name: name, Driver: driver}, nil
}

func (d *DockerService) RemoveNetwork(ctx context.Context, id string) error {
	return d.cli.NetworkRemove(ctx, id)
}

// ---- Stacks (Compose) ----

type StackInfo struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Services    int               `json:"services"`
	ComposeYAML string            `json:"compose_yaml,omitempty"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
	CreatedAt   string            `json:"created_at"`
}

func (d *DockerService) ListStacks(ctx context.Context) ([]StackInfo, error) {
	rows, err := d.db.Query(ctx,
		`SELECT id, name, status, compose_yaml, env_vars, created_at FROM docker_stacks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stacks []StackInfo
	for rows.Next() {
		var s StackInfo
		var envJSON []byte
		if err := rows.Scan(&s.ID, &s.Name, &s.Status, &s.ComposeYAML, &envJSON, &s.CreatedAt); err != nil {
			continue
		}
		// Count services by counting lines with "image:" or "build:" in YAML
		s.Services = strings.Count(s.ComposeYAML, "image:") + strings.Count(s.ComposeYAML, "build:")
		stacks = append(stacks, s)
	}
	return stacks, nil
}

type DeployStackRequest struct {
	Name        string            `json:"name"`
	ComposeYAML string            `json:"compose_yaml"`
	EnvVars     map[string]string `json:"env_vars,omitempty"`
}

func (d *DockerService) DeployStack(ctx context.Context, req DeployStackRequest, userID string) (*StackInfo, error) {
	// Save to DB
	var id, createdAt string
	err := d.db.QueryRow(ctx,
		`INSERT INTO docker_stacks (name, compose_yaml, env_vars, status, user_id)
		 VALUES ($1, $2, $3, 'deploying', $4)
		 ON CONFLICT (name) DO UPDATE SET compose_yaml = $2, env_vars = $3, status = 'deploying', updated_at = NOW()
		 RETURNING id, created_at::text`,
		req.Name, req.ComposeYAML, req.EnvVars, userID,
	).Scan(&id, &createdAt)
	if err != nil {
		return nil, fmt.Errorf("save stack: %w", err)
	}

	// We mark it deployed. Actual docker compose up would require docker compose CLI.
	// For the MVP, we store the stack and the user can deploy via terminal.
	d.db.Exec(ctx, `UPDATE docker_stacks SET status = 'deployed' WHERE id = $1`, id)

	return &StackInfo{
		ID:        id,
		Name:      req.Name,
		Status:    "deployed",
		CreatedAt: createdAt,
	}, nil
}

func (d *DockerService) RemoveStack(ctx context.Context, name string) error {
	_, err := d.db.Exec(ctx, `DELETE FROM docker_stacks WHERE name = $1`, name)
	return err
}

// ---- Stats ----

type DockerStats struct {
	Containers int `json:"containers"`
	Running    int `json:"running"`
	Stopped    int `json:"stopped"`
	Images     int `json:"images"`
	Volumes    int `json:"volumes"`
	Networks   int `json:"networks"`
}

func (d *DockerService) GetStats(ctx context.Context) (*DockerStats, error) {
	info, err := d.cli.Info(ctx)
	if err != nil {
		return nil, err
	}

	vols, _ := d.cli.VolumeList(ctx, volume.ListOptions{})
	volCount := 0
	if vols.Volumes != nil {
		volCount = len(vols.Volumes)
	}

	nets, _ := d.cli.NetworkList(ctx, network.ListOptions{
		Filters: filters.NewArgs(filters.Arg("type", "custom")),
	})

	return &DockerStats{
		Containers: info.Containers,
		Running:    info.ContainersRunning,
		Stopped:    info.ContainersStopped,
		Images:     info.Images,
		Volumes:    volCount,
		Networks:   len(nets) + 3, // +3 for default bridge, host, none
	}, nil
}

// ---- Container Stats (live CPU/RAM/Net/IO) ----

type ContainerStatsResult struct {
	CPUPercent float64 `json:"cpu_percent"`
	MemUsage   int64   `json:"mem_usage"`
	MemLimit   int64   `json:"mem_limit"`
	MemPercent float64 `json:"mem_percent"`
	NetRx      int64   `json:"net_rx"`
	NetTx      int64   `json:"net_tx"`
	BlockRead  int64   `json:"block_read"`
	BlockWrite int64   `json:"block_write"`
	Pids       int64   `json:"pids"`
}

func (d *DockerService) ContainerStats(ctx context.Context, id string) (*ContainerStatsResult, error) {
	resp, err := d.cli.ContainerStats(ctx, id, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, err
	}

	result := &ContainerStatsResult{Pids: int64(stats.PidsStats.Current)}

	// CPU
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	sysDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)
	if sysDelta > 0 && cpuDelta > 0 {
		cpuCount := float64(stats.CPUStats.OnlineCPUs)
		if cpuCount == 0 {
			cpuCount = float64(len(stats.CPUStats.CPUUsage.PercpuUsage))
		}
		result.CPUPercent = (cpuDelta / sysDelta) * cpuCount * 100.0
	}

	// Memory
	result.MemUsage = int64(stats.MemoryStats.Usage)
	result.MemLimit = int64(stats.MemoryStats.Limit)
	if result.MemLimit > 0 {
		result.MemPercent = float64(result.MemUsage) / float64(result.MemLimit) * 100.0
	}

	// Network
	for _, n := range stats.Networks {
		result.NetRx += int64(n.RxBytes)
		result.NetTx += int64(n.TxBytes)
	}

	// Block IO
	for _, bio := range stats.BlkioStats.IoServiceBytesRecursive {
		switch bio.Op {
		case "read", "Read":
			result.BlockRead += int64(bio.Value)
		case "write", "Write":
			result.BlockWrite += int64(bio.Value)
		}
	}

	return result, nil
}

// ---- Container Rename ----

func (d *DockerService) RenameContainer(ctx context.Context, id, newName string) error {
	return d.cli.ContainerRename(ctx, id, newName)
}

// ---- Container Duplicate ----

func (d *DockerService) DuplicateContainer(ctx context.Context, id, newName string) (string, error) {
	inspect, err := d.cli.ContainerInspect(ctx, id)
	if err != nil {
		return "", err
	}

	config := inspect.Config
	hostConfig := inspect.HostConfig

	resp, err := d.cli.ContainerCreate(ctx, config, hostConfig, nil, nil, newName)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// ---- Container Processes ----

type ProcessList struct {
	Titles    []string   `json:"titles"`
	Processes [][]string `json:"processes"`
}

func (d *DockerService) ContainerProcesses(ctx context.Context, id string) (*ProcessList, error) {
	top, err := d.cli.ContainerTop(ctx, id, nil)
	if err != nil {
		return nil, err
	}
	return &ProcessList{Titles: top.Titles, Processes: top.Processes}, nil
}

// ---- Image History ----

type ImageHistoryItem struct {
	ID        string `json:"id"`
	Created   int64  `json:"created"`
	CreatedBy string `json:"created_by"`
	Size      int64  `json:"size"`
	Comment   string `json:"comment"`
}

func (d *DockerService) ImageHistory(ctx context.Context, id string) ([]ImageHistoryItem, error) {
	history, err := d.cli.ImageHistory(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]ImageHistoryItem, 0, len(history))
	for _, h := range history {
		hID := h.ID
		if strings.HasPrefix(hID, "sha256:") && len(hID) > 19 {
			hID = hID[7:19]
		}
		out = append(out, ImageHistoryItem{
			ID:        hID,
			Created:   h.Created,
			CreatedBy: h.CreatedBy,
			Size:      h.Size,
			Comment:   h.Comment,
		})
	}
	return out, nil
}

// ---- Image Tag ----

func (d *DockerService) TagImage(ctx context.Context, source, target string) error {
	return d.cli.ImageTag(ctx, source, target)
}

// ---- Network Connect/Disconnect ----

func (d *DockerService) NetworkConnect(ctx context.Context, networkID, containerID string) error {
	return d.cli.NetworkConnect(ctx, networkID, containerID, nil)
}

func (d *DockerService) NetworkDisconnect(ctx context.Context, networkID, containerID string, force bool) error {
	return d.cli.NetworkDisconnect(ctx, networkID, containerID, force)
}

// ---- System Info ----

type DockerSystemInfo struct {
	DockerVersion string `json:"docker_version"`
	APIVersion    string `json:"api_version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	Kernel        string `json:"kernel"`
	CPUs          int    `json:"cpus"`
	Memory        int64  `json:"memory"`
	StorageDriver string `json:"storage_driver"`
	RootDir       string `json:"root_dir"`
	Containers    int    `json:"containers"`
	Images        int    `json:"images"`
	Runtime       string `json:"runtime"`
}

func (d *DockerService) SystemInfo(ctx context.Context) (*DockerSystemInfo, error) {
	info, err := d.cli.Info(ctx)
	if err != nil {
		return nil, err
	}
	ver, _ := d.cli.ServerVersion(ctx)

	runtime := "runc"
	if info.DefaultRuntime != "" {
		runtime = info.DefaultRuntime
	}

	return &DockerSystemInfo{
		DockerVersion: ver.Version,
		APIVersion:    ver.APIVersion,
		OS:            info.OperatingSystem,
		Arch:          info.Architecture,
		Kernel:        info.KernelVersion,
		CPUs:          info.NCPU,
		Memory:        info.MemTotal,
		StorageDriver: info.Driver,
		RootDir:       info.DockerRootDir,
		Containers:    info.Containers,
		Images:        info.Images,
		Runtime:       runtime,
	}, nil
}

// ---- Prune ----

type PruneResult struct {
	ContainersDeleted int   `json:"containers_deleted"`
	ImagesDeleted     int   `json:"images_deleted"`
	VolumesDeleted    int   `json:"volumes_deleted"`
	NetworksDeleted   int   `json:"networks_deleted"`
	SpaceReclaimed    int64 `json:"space_reclaimed"`
}

func (d *DockerService) PruneSystem(ctx context.Context) (*PruneResult, error) {
	result := &PruneResult{}

	c, _ := d.cli.ContainersPrune(ctx, filters.Args{})
	result.ContainersDeleted = len(c.ContainersDeleted)
	result.SpaceReclaimed += int64(c.SpaceReclaimed)

	i, _ := d.cli.ImagesPrune(ctx, filters.Args{})
	result.ImagesDeleted = len(i.ImagesDeleted)
	result.SpaceReclaimed += int64(i.SpaceReclaimed)

	v, _ := d.cli.VolumesPrune(ctx, filters.Args{})
	result.VolumesDeleted = len(v.VolumesDeleted)
	result.SpaceReclaimed += int64(v.SpaceReclaimed)

	n, _ := d.cli.NetworksPrune(ctx, filters.Args{})
	result.NetworksDeleted = len(n.NetworksDeleted)

	return result, nil
}

// ---- Expose Docker Client (for exec handler) ----

func (d *DockerService) Client() *client.Client {
	return d.cli
}

// ---- Pause / Unpause ----

func (d *DockerService) PauseContainer(ctx context.Context, id string) error {
	return d.cli.ContainerPause(ctx, id)
}

func (d *DockerService) UnpauseContainer(ctx context.Context, id string) error {
	return d.cli.ContainerUnpause(ctx, id)
}

// ---- Kill Container ----

func (d *DockerService) KillContainer(ctx context.Context, id, signal string) error {
	if signal == "" {
		signal = "SIGKILL"
	}
	return d.cli.ContainerKill(ctx, id, signal)
}

// ---- Commit Container (export to image) ----

type CommitResult struct {
	ImageID string `json:"image_id"`
}

func (d *DockerService) CommitContainer(ctx context.Context, containerID, repo, tag, comment string) (*CommitResult, error) {
	resp, err := d.cli.ContainerCommit(ctx, containerID, container.CommitOptions{
		Reference: repo + ":" + tag,
		Comment:   comment,
	})
	if err != nil {
		return nil, err
	}
	return &CommitResult{ImageID: resp.ID}, nil
}

// ---- Container File Browser ----

type FileEntry struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime int64  `json:"mod_time"`
	Mode    string `json:"mode"`
}

func (d *DockerService) BrowseContainerFiles(ctx context.Context, id, dirPath string) ([]FileEntry, error) {
	if dirPath == "" {
		dirPath = "/"
	}

	reader, stat, err := d.cli.CopyFromContainer(ctx, id, dirPath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	// If it's a single file, return just that entry
	if !stat.Mode.IsDir() {
		return []FileEntry{{
			Name:    stat.Name,
			Path:    dirPath,
			Size:    stat.Size,
			IsDir:   false,
			ModTime: stat.Mtime.Unix(),
			Mode:    stat.Mode.String(),
		}}, nil
	}

	// Read tar archive to list directory contents
	tr := tar.NewReader(reader)
	entries := []FileEntry{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}

		// Only include direct children, not nested entries
		name := strings.TrimPrefix(hdr.Name, path.Base(dirPath)+"/")
		if name == "" || name == "." {
			continue
		}
		// Skip deeper nested entries
		clean := strings.TrimSuffix(name, "/")
		if strings.Contains(clean, "/") {
			continue
		}

		fullPath := dirPath
		if !strings.HasSuffix(fullPath, "/") {
			fullPath += "/"
		}
		fullPath += clean

		entries = append(entries, FileEntry{
			Name:    clean,
			Path:    fullPath,
			Size:    hdr.Size,
			IsDir:   hdr.Typeflag == tar.TypeDir,
			ModTime: hdr.ModTime.Unix(),
			Mode:    hdr.FileInfo().Mode().String(),
		})
	}

	return entries, nil
}

// ---- Image Build from Dockerfile ----

type BuildResult struct {
	Logs string `json:"logs"`
}

func (d *DockerService) BuildImage(ctx context.Context, dockerfile, tag string) (*BuildResult, error) {
	// Create a tar archive with the Dockerfile
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	header := &tar.Header{
		Name: "Dockerfile",
		Mode: 0644,
		Size: int64(len(dockerfile)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return nil, err
	}
	if _, err := tw.Write([]byte(dockerfile)); err != nil {
		return nil, err
	}
	tw.Close()

	resp, err := d.cli.ImageBuild(ctx, buf, build.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read build output
	output, _ := io.ReadAll(resp.Body)
	return &BuildResult{Logs: string(output)}, nil
}

// ---- Push Image to Registry ----

func (d *DockerService) PushImage(ctx context.Context, ref string) (string, error) {
	reader, err := d.cli.ImagePush(ctx, ref, image.PushOptions{
		RegistryAuth: "e30=", // base64 empty JSON {} for public/local registries
	})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	output, _ := io.ReadAll(reader)
	return string(output), nil
}

// ---- Docker Events Stream ----

type DockerEvent struct {
	Type   string `json:"type"`
	Action string `json:"action"`
	Actor  string `json:"actor"`
	Time   int64  `json:"time"`
}

func (d *DockerService) StreamEvents(ctx context.Context, since time.Duration) ([]DockerEvent, error) {
	sinceTime := time.Now().Add(-since)
	untilTime := time.Now()

	msgChan, errChan := d.cli.Events(ctx, events.ListOptions{
		Since: sinceTime.Format(time.RFC3339),
		Until: untilTime.Format(time.RFC3339),
	})

	var result []DockerEvent
	for {
		select {
		case msg, ok := <-msgChan:
			if !ok {
				return result, nil
			}
			actorName := msg.Actor.ID
			if n, ok := msg.Actor.Attributes["name"]; ok {
				actorName = n
			}
			result = append(result, DockerEvent{
				Type:   string(msg.Type),
				Action: string(msg.Action),
				Actor:  actorName,
				Time:   msg.Time,
			})
		case err := <-errChan:
			if err != nil {
				return result, err
			}
			return result, nil
		}
	}
}

