package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/novapanel/novapanel/internal/agent"
)

func main() {
	apiURL := os.Getenv("NOVAPANEL_API_URL")
	token := os.Getenv("NOVAPANEL_AGENT_TOKEN")
	serverID := os.Getenv("NOVAPANEL_SERVER_ID")

	if apiURL == "" || serverID == "" {
		log.Fatal("NOVAPANEL_API_URL and NOVAPANEL_SERVER_ID are required environment variables")
	}

	client := agent.NewClient(apiURL, token, serverID)
	log.Printf("Starting NovaPanel Agent for Server ID: %s", serverID)
	log.Printf("Connecting to API: %s", apiURL)

	// Send an initial heartbeat immediately
	sendMetrics(client, serverID)

	ticker := time.NewTicker(30 * time.Second)
	quit := make(chan struct{})

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-ticker.C:
				sendMetrics(client, serverID)
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	<-sigs
	log.Println("Shutting down NovaPanel agent gracefully...")
	close(quit)
}

func sendMetrics(client *agent.Client, serverID string) {
	metrics, err := agent.CollectMetrics(serverID)
	if err != nil {
		log.Printf("Error collecting metrics: %v", err)
		return
	}

	if err := client.SendHeartbeat(metrics); err != nil {
		log.Printf("Error sending heartbeat: %v", err)
	} else {
		log.Printf("Heartbeat sent successfully (CPU: %.1f%%, RAM: %dMB/%dMB, Disk: %.1fGB/%.1fGB)", 
			metrics.CPUPercent, metrics.RAMUsedMB, metrics.RAMTotalMB, metrics.DiskUsedGB, metrics.DiskTotalGB)
	}
}
