package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"
	"time"
)

type DepricatedUptimeConfig struct {
	APIKey         string `env:"UPTIMEROBOT_API_KEY"`
	UpdateInterval int    `env:"UPTIMEROBOT_UPDATE_INTERVAL" envDefault:"300"`
}

type UptimeConfig struct {
	Provider       string `env:"UPTIME_PROVIDER" envDefault:"uptimerobot"`
	APIKey         string
	UpdateInterval int `env:"UPTIME_UPDATE_INTERVAL" envDefault:"300"`
}

type UptimeSite struct {
	FriendlyName string
	Url          string
	Up           bool
}

type UptimeManager struct {
	provider       string
	sites          []UptimeSite
	lastUpdate     time.Time
	mutex          sync.RWMutex
	updateChan     chan struct{}
	updateInterval int
	apiKey         string
}

func NewUptimeManager(config *UptimeConfig) *UptimeManager {
	if config.APIKey == "" {
		log.Fatalln("An API Key is required to use Uptime Monitoring!")
		return nil
	}

	updateInterval := config.UpdateInterval
	if updateInterval < 1 {
		updateInterval = 300
	}

	uptimeManager := &UptimeManager{
		provider:       config.Provider,
		updateChan:     make(chan struct{}),
		updateInterval: updateInterval,
		apiKey:         config.APIKey,
		sites:          []UptimeSite{},
	}

	go uptimeManager.updateWorker()

	uptimeManager.updateChan <- struct{}{}

	return uptimeManager
}

func (u *UptimeManager) GetUptime() []UptimeSite {
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

type UptimeRobotSite struct {
	FriendlyName string `json:"friendly_name"`
	Url          string `json:"url"`
	Status       int    `json:"status"`
}

type UptimeRobotResponse struct {
	Monitors []UptimeRobotSite `json:"monitors"`
}

type BetterUptimeSite struct {
	MonitorType string `json:"type"`
	Attributes  struct {
		PronounceableName string `json:"pronounceable_name"`
		Url               string `json:"url"`
		Status            string `json:"status"`
	} `json:"attributes"`
}

type BetterUptimeResponse struct {
	Monitors []BetterUptimeSite `json:"data"`
}

func (u *UptimeManager) update() {
	var monitors []UptimeSite
	switch u.provider {
	case "uptimerobot":
		monitors = u.updateUptimeRobot()
	case "betteruptime":
		monitors = u.updateBetterUptime()
	default:
		log.Fatalln("Invalid Uptime Provider!")
	}

	u.mutex.Lock()
	u.sites = monitors
	u.lastUpdate = time.Now()
	u.mutex.Unlock()
}

func (u *UptimeManager) updateUptimeRobot() []UptimeSite {
	resp, err := http.Post("https://api.uptimerobot.com/v2/getMonitors?api_key="+u.apiKey, "application/json", nil)
	if err != nil {
		fmt.Printf("Error fetching uptime data: %v\n", err)
		return []UptimeSite{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return []UptimeSite{}
	}

	var rawMonitors UptimeRobotResponse
	if err := json.Unmarshal(body, &rawMonitors); err != nil {
		fmt.Printf("Error parsing uptime data: %v\n", err)
		return []UptimeSite{}
	}

	var monitors []UptimeSite
	for _, rawMonitor := range rawMonitors.Monitors {
		monitors = append(monitors, UptimeSite{
			FriendlyName: rawMonitor.FriendlyName,
			Url:          rawMonitor.Url,
			Up:           rawMonitor.Status == 2,
		})
	}

	return monitors
}

func (u *UptimeManager) updateBetterUptime() []UptimeSite {
	client := &http.Client{}
	req, err := http.NewRequest("GET", "https://uptime.betterstack.com/api/v2/monitors", nil)
	if err != nil {
		fmt.Printf("Error fetching uptime data: %v\n", err)
		return []UptimeSite{}
	}
	req.Header.Add("Authorization", "Bearer "+u.apiKey)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error fetching uptime data: %v\n", err)
		return []UptimeSite{}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return []UptimeSite{}
	}

	var rawMonitors BetterUptimeResponse
	if err := json.Unmarshal(body, &rawMonitors); err != nil {
		fmt.Printf("Error parsing uptime data: %v\n", err)
		return []UptimeSite{}
	}

	// alphabetically sort the monitors because UptimeRobot does, but BetterUptime doesnt (or sorts by something else?), and I want them to be consistent
	sort.Slice(rawMonitors.Monitors, func(i, j int) bool {
		return rawMonitors.Monitors[i].Attributes.PronounceableName < rawMonitors.Monitors[j].Attributes.PronounceableName
	})

	var monitors []UptimeSite
	for _, rawMonitor := range rawMonitors.Monitors {
		monitors = append(monitors, UptimeSite{
			FriendlyName: rawMonitor.Attributes.PronounceableName,
			Url:          rawMonitor.Attributes.Url,
			Up:           rawMonitor.Attributes.Status == "up",
		})
	}

	return monitors
}
