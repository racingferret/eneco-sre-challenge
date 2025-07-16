package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"time"
)

// Alert defines the structure of an individual alert
type Alert struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Service     string    `json:"service"`
	Component   string    `json:"component"`
	Severity    string    `json:"severity"`
	Metric      string    `json:"metric"`
	Value       int       `json:"value"`
	Threshold   int       `json:"threshold"`
	Description string    `json:"description"`
}

// AlertGroup groups related alerts by service and component, with total priority
type AlertGroup struct {
	ServiceName string
	Component   string
	Alerts      []Alert
	Priority    int
}

// calculatePriority assigns a score based on severity
func calculatePriority(severity string) int {
	switch severity {
	case "critical":
		return 10
	case "warning":
		return 5
	case "info":
		return 1
	default:
		return 0
	}
}

// calculateDeviationPercent computes the % deviation from threshold
func calculateDeviationPercent(value, threshold int) float64 {
	if threshold == 0 {
		return 0
	}
	return (float64(value-threshold) / float64(threshold)) * 100
}

// readAlertsFromFile loads alerts from a JSON file
func readAlertsFromFile(filename string) ([]Alert, error) {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var data struct {
		Alerts []Alert `json:"alerts"`
	}

	err = json.Unmarshal(fileData, &data)
	if err != nil {
		return nil, err
	}

	return data.Alerts, nil
}

// filterAlerts applies severity, time range, service, or "last X minutes" filters
func filterAlerts(alerts []Alert, severity string, startTime, endTime time.Time, service string, lastXMinutes int) []Alert {
	var filtered []Alert
	cutoff := time.Time{}
	if lastXMinutes > 0 {
		cutoff = time.Now().Add(-time.Duration(lastXMinutes) * time.Minute)
	}

	for _, alert := range alerts {
		if severity != "" && alert.Severity != severity {
			continue
		}
		if !startTime.IsZero() && !endTime.IsZero() {
			if alert.Timestamp.Before(startTime) || alert.Timestamp.After(endTime) {
				continue
			}
		}
		if lastXMinutes > 0 && alert.Timestamp.Before(cutoff) {
			continue
		}
		if service != "" && alert.Service != service {
			continue
		}
		filtered = append(filtered, alert)
	}
	return filtered
}

// groupAlertsByServiceAndComponent clusters alerts and sums priorities
func groupAlertsByServiceAndComponent(alerts []Alert) []AlertGroup {
	alertGroups := make(map[string]map[string]*AlertGroup)

	for _, alert := range alerts {
		priority := calculatePriority(alert.Severity)

		if _, ok := alertGroups[alert.Service]; !ok {
			alertGroups[alert.Service] = make(map[string]*AlertGroup)
		}

		if group, ok := alertGroups[alert.Service][alert.Component]; ok {
			group.Alerts = append(group.Alerts, alert)
			group.Priority += priority
		} else {
			alertGroups[alert.Service][alert.Component] = &AlertGroup{
				ServiceName: alert.Service,
				Component:   alert.Component,
				Alerts:      []Alert{alert},
				Priority:    priority,
			}
		}
	}

	var result []AlertGroup
	for _, components := range alertGroups {
		for _, group := range components {
			result = append(result, *group)
		}
	}
	return result
}

func printHelp() {
	fmt.Println(`Usage: alerts-analyzer [options]

Options:
  --file <filename>        Path to JSON file (default: sample_alerts.json)
  --severity <level>       Filter by severity: critical, warning, info
  --start <time>           Start time (RFC3339 format)
  --end <time>             End time (RFC3339 format)
  --last <minutes>         Show alerts from the last X minutes
  --service <name>         Filter by service name
  --help                   Show this help message`)
}

func main() {
	// Flags
	filename := flag.String("file", "sample_alerts.json", "Path to JSON file")
	severity := flag.String("severity", "", "Severity filter (critical, warning, info)")
	start := flag.String("start", "", "Start time (RFC3339)")
	end := flag.String("end", "", "End time (RFC3339)")
	lastMinutes := flag.Int("last", 0, "Show alerts from last X minutes")
	serviceFilter := flag.String("service", "", "Filter by service name")
	help := flag.Bool("help", false, "Show help")
	flag.Parse()

	if *help {
		printHelp()
		return
	}

	// Parse optional time range
	var startTime, endTime time.Time
	var err error
	if *start != "" {
		startTime, err = time.Parse(time.RFC3339, *start)
		if err != nil {
			log.Fatalf("Invalid start time format: %v", err)
		}
	}
	if *end != "" {
		endTime, err = time.Parse(time.RFC3339, *end)
		if err != nil {
			log.Fatalf("Invalid end time format: %v", err)
		}
	}

	// Load alerts
	alerts, err := readAlertsFromFile(*filename)
	if err != nil {
		log.Fatalf("Error reading or parsing JSON: %v", err)
	}
	fmt.Println("JSON format OK!")

	// Apply filters
	filtered := filterAlerts(alerts, *severity, startTime, endTime, *serviceFilter, *lastMinutes)

	// Group and sort alerts
	grouped := groupAlertsByServiceAndComponent(filtered)
	sort.Slice(grouped, func(i, j int) bool {
		return grouped[i].Priority > grouped[j].Priority
	})

	// Group AlertGroups by service
	serviceMap := make(map[string][]AlertGroup)
	orderedServices := []string{}
	seen := make(map[string]bool)

	for _, group := range grouped {
		serviceMap[group.ServiceName] = append(serviceMap[group.ServiceName], group)
		if !seen[group.ServiceName] {
			seen[group.ServiceName] = true
			orderedServices = append(orderedServices, group.ServiceName)
		}
	}

	// Output
	fmt.Println("\nGrouped Alerts by Service (Ordered by Total Priority):")
	for _, service := range orderedServices {
		fmt.Printf("\nService: %s\n", service)
		for _, group := range serviceMap[service] {
			fmt.Printf("  Component: %s, Total Priority: %d\n", group.Component, group.Priority)
			for _, alert := range group.Alerts {
				deviation := calculateDeviationPercent(alert.Value, alert.Threshold)
				fmt.Printf("    - ID: %s | Severity: %s | Time: %s | Metric: %s | Value: %d | Threshold: %d | Deviation: %.2f%% | Description: %s\n",
					alert.ID, alert.Severity, alert.Timestamp.Format(time.RFC3339),
					alert.Metric, alert.Value, alert.Threshold, deviation, alert.Description)
			}
		}
	}
}
