package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/version"
	"github.com/novapanel/novapanel/internal/websocket"
)

const (
	githubReleasesURL = "https://api.github.com/repos/codepromaxtech/novapanel/releases/latest"
	apiImage          = "codepromax24/novapanel-api:latest"
	automationImage   = "codepromax24/novapanel-automation:latest"
	apiContainer      = "novapanel-api"
	automationContainer = "novapanel-automation"
)

// UpdateStatus is returned to the frontend.
type UpdateStatus struct {
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version"`
	UpdateAvailable bool   `json:"update_available"`
	ReleaseNotes    string `json:"release_notes"`
	ReleaseURL      string `json:"release_url"`
	LastChecked     string `json:"last_checked,omitempty"`
	Applying        bool   `json:"applying"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
}

// UpdateService checks GitHub for new releases and applies Docker-based updates.
type UpdateService struct {
	pool      *pgxpool.Pool
	dockerCli *client.Client
	wsHub     *websocket.Hub
	applying  bool
}

func NewUpdateService(pool *pgxpool.Pool, dockerCli *client.Client, wsHub *websocket.Hub) *UpdateService {
	return &UpdateService{pool: pool, dockerCli: dockerCli, wsHub: wsHub}
}

// GetStatus returns the cached update status from system_settings.
func (s *UpdateService) GetStatus(ctx context.Context) UpdateStatus {
	status := UpdateStatus{
		CurrentVersion: version.AppVersion,
		Applying:       s.applying,
	}

	rows, err := s.pool.Query(ctx,
		`SELECT key, value FROM system_settings WHERE key LIKE 'update_%'`)
	if err != nil {
		return status
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		settings[k] = v
	}

	status.LatestVersion = settings["update_latest_version"]
	status.ReleaseNotes = settings["update_release_notes"]
	status.ReleaseURL = settings["update_release_url"]
	status.LastChecked = settings["update_last_check"]
	status.UpdateAvailable = settings["update_available"] == "true"
	return status
}

// CheckForUpdates fetches the latest GitHub release and updates system_settings.
func (s *UpdateService) CheckForUpdates(ctx context.Context) (UpdateStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, githubReleasesURL, nil)
	if err != nil {
		return UpdateStatus{}, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "NovaPanel/"+version.AppVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return UpdateStatus{}, fmt.Errorf("github API error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// No releases published yet
		return s.saveStatus(ctx, version.AppVersion, "", "", false), nil
	}
	if resp.StatusCode != http.StatusOK {
		return UpdateStatus{}, fmt.Errorf("github API returned %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return UpdateStatus{}, fmt.Errorf("parse github response: %w", err)
	}

	latest := normalizeTag(release.TagName)
	current := normalizeTag(version.AppVersion)
	available := latest != "" && latest != current && isNewer(latest, current)

	status := s.saveStatus(ctx, release.TagName, release.Body, release.HTMLURL, available)

	if available {
		s.wsHub.BroadcastEvent("update:available", status)
	}

	return status, nil
}

func (s *UpdateService) saveStatus(ctx context.Context, latestTag, notes, url string, available bool) UpdateStatus {
	now := time.Now().UTC().Format(time.RFC3339)
	availStr := "false"
	if available {
		availStr = "true"
	}

	upsert := func(k, v string) {
		s.pool.Exec(ctx,
			`INSERT INTO system_settings (key, value, encrypted) VALUES ($1, $2, false)
			 ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()`,
			k, v)
	}
	upsert("update_latest_version", latestTag)
	upsert("update_release_notes", notes)
	upsert("update_release_url", url)
	upsert("update_last_check", now)
	upsert("update_available", availStr)

	return UpdateStatus{
		CurrentVersion:  version.AppVersion,
		LatestVersion:   latestTag,
		ReleaseNotes:    notes,
		ReleaseURL:      url,
		UpdateAvailable: available,
		LastChecked:     now,
		Applying:        s.applying,
	}
}

// ApplyUpdate pulls the latest images and restarts containers.
// It returns immediately; progress is streamed via WebSocket (update:progress / update:complete / update:error).
func (s *UpdateService) ApplyUpdate(ctx context.Context) error {
	if s.applying {
		return fmt.Errorf("update already in progress")
	}
	s.applying = true

	go func() {
		defer func() { s.applying = false }()
		bgCtx := context.Background()

		s.broadcast("pulling", "Pulling novapanel-api image...", 10)
		if err := s.pullImage(bgCtx, apiImage); err != nil {
			s.broadcastError(fmt.Sprintf("Failed to pull %s: %v", apiImage, err))
			return
		}
		s.broadcast("pulling", "Pulling novapanel-automation image...", 40)
		if err := s.pullImage(bgCtx, automationImage); err != nil {
			s.broadcastError(fmt.Sprintf("Failed to pull %s: %v", automationImage, err))
			return
		}

		s.broadcast("restarting", "Restarting automation service...", 70)
		if err := s.dockerCli.ContainerRestart(bgCtx, automationContainer, container.StopOptions{}); err != nil {
			log.Printf("Warning: restart automation: %v", err)
		}

		s.broadcast("restarting", "Applying update — NovaPanel will restart in 3 seconds...", 90)
		s.wsHub.BroadcastEvent("update:complete", map[string]interface{}{
			"message": "Update applied. Reconnecting...",
		})

		// Restart self last so the response is delivered first
		go func() {
			time.Sleep(3 * time.Second)
			if err := s.dockerCli.ContainerRestart(context.Background(), apiContainer, container.StopOptions{}); err != nil {
				log.Printf("Warning: restart api container: %v", err)
			}
		}()
	}()

	return nil
}

func (s *UpdateService) pullImage(ctx context.Context, ref string) error {
	reader, err := s.dockerCli.ImagePull(ctx, ref, dockerimage.PullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	// Drain the reader so the pull completes; broadcast layer progress
	dec := json.NewDecoder(reader)
	for {
		var event map[string]interface{}
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if status, ok := event["status"].(string); ok {
			if id, ok := event["id"].(string); ok && id != "" {
				s.wsHub.BroadcastEvent("update:layer", map[string]string{
					"image": ref, "layer": id, "status": status,
				})
			}
		}
	}
	return nil
}

func (s *UpdateService) broadcast(step, msg string, pct int) {
	s.wsHub.BroadcastEvent("update:progress", map[string]interface{}{
		"step":    step,
		"message": msg,
		"percent": pct,
	})
}

func (s *UpdateService) broadcastError(msg string) {
	s.wsHub.BroadcastEvent("update:error", map[string]string{"message": msg})
	log.Printf("Update error: %s", msg)
}

// RunBackgroundChecker periodically checks for updates every 6 hours.
func (s *UpdateService) RunBackgroundChecker(ctx context.Context) {
	// Initial check on startup (with a small delay to let the DB settle)
	time.Sleep(30 * time.Second)
	if _, err := s.CheckForUpdates(ctx); err != nil {
		log.Printf("Initial update check: %v", err)
	}

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if _, err := s.CheckForUpdates(ctx); err != nil {
				log.Printf("Periodic update check: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

func normalizeTag(tag string) string {
	return strings.TrimPrefix(strings.TrimSpace(tag), "v")
}

// isNewer returns true if latest > current using simple semver comparison.
func isNewer(latest, current string) bool {
	lp := parseSemver(latest)
	cp := parseSemver(current)
	for i := range lp {
		if i >= len(cp) {
			return lp[i] > 0
		}
		if lp[i] > cp[i] {
			return true
		}
		if lp[i] < cp[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	var major, minor, patch int
	fmt.Sscanf(v, "%d.%d.%d", &major, &minor, &patch)
	return [3]int{major, minor, patch}
}
