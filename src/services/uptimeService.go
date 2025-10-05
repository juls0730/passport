package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type DepricatedUptimeConfig struct {
	APIKey         string `env:"UPTIMEROBOT_API_KEY"`
	UpdateInterval int    `env:"UPTIMEROBOT_UPDATE_INTERVAL" envDefault:"300"`
}

type UptimeConfig struct {
	APIKey         string
	UpdateInterval int `env:"UPTIME_UPDATE_INTERVAL" envDefault:"300"`
}

type UptimeRobotSite struct {
	FriendlyName string `json:"friendly_name"`
	Url          string `json:"url"`
	Status       int    `json:"status"`
	Up           bool   `json:"-"`
}

type UptimeManager struct {
	sites          []UptimeRobotSite
	lastUpdate     time.Time
	mutex          sync.RWMutex
	updateChan     chan struct{}
	updateInterval int
	apiKey         string
}

func NewUptimeManager(config *UptimeConfig) *UptimeManager {
	if config.APIKey == "" {
		log.Fatalln("UptimeRobot API Key is required!")
		return nil
	}

	updateInterval := config.UpdateInterval
	if updateInterval < 1 {
		updateInterval = 300
	}

	uptimeManager := &UptimeManager{
		updateChan:     make(chan struct{}),
		updateInterval: updateInterval,
		apiKey:         config.APIKey,
		sites:          []UptimeRobotSite{},
	}

	go uptimeManager.updateWorker()

	uptimeManager.updateChan <- struct{}{}

	return uptimeManager
}

func (u *UptimeManager) GetUptime() []UptimeRobotSite {
	u.mutex.RLock()
	defer u.mutex.RUnlock()
	return u.sites
}

func (u *UptimeManager) updateWorker() {
	ticker := time.NewTicker(time.Duration(u.updateInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-u.updateChan:
			u.update()
		case <-ticker.C:
			u.update()
		}
	}
}

type UptimeRobotResponse struct {
	Monitors []UptimeRobotSite `json:"monitors"`
}

func (u *UptimeManager) update() {
	resp, err := http.Post("https://api.uptimerobot.com/v2/getMonitors?api_key="+u.apiKey, "application/json", nil)
	if err != nil {
		fmt.Printf("Error fetching uptime data: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}

	var monitors UptimeRobotResponse
	if err := json.Unmarshal(body, &monitors); err != nil {
		fmt.Printf("Error parsing uptime data: %v\n", err)
		return
	}

	for i, monitor := range monitors.Monitors {
		monitors.Monitors[i].Up = monitor.Status == 2
	}

	u.mutex.Lock()
	u.sites = monitors.Monitors
	u.lastUpdate = time.Now()
	u.mutex.Unlock()
}
