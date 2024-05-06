package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"
	"time"

	"golang.org/x/net/websocket"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LokiAddress          string `yaml:"loki_address"`
	LokiWebSocketAddress string `yaml:"loki_websocket_address"`
	LokiLabelKey         string `yaml:"loki_label_key"`
}

type LokiQueryTailResponse struct {
	Streams []struct {
		Stream map[string]string `json:"stream"`
		Values [][]interface{}   `json:"values"`
	} `json:"streams"`
	DroppedEntries []struct {
		Labels    map[string]string `json:"labels"`
		Timestamp string            `json:"timestamp"`
	} `json:"dropped_entries"`
}

type PodInfo struct {
	Namespace string
	PodName   string
	StartTime time.Time
}

func getTailLogsFromLoki(podInfo PodInfo, lokiAddress, LokiWebsocketAddress string, config Config) error {
	startedAt := podInfo.StartTime.UnixNano()

	query := fmt.Sprintf(`{%s="%s"}`, config.LokiLabelKey, podInfo.PodName)

	u, err := url.Parse(fmt.Sprintf("%s/loki/api/v1/tail", LokiWebsocketAddress))
	if err != nil {
		return err
	}

	params := u.Query()
	params.Set("query", query)
	params.Set("start", strconv.FormatInt(startedAt, 10))
	u.RawQuery = params.Encode()

	wsConfig, err := websocket.NewConfig(u.String(), lokiAddress)
	if err != nil {
		return err
	}
	wsConfig.Header.Set("X-Scope-OrgID", podInfo.Namespace)

	ws, err := websocket.DialConfig(wsConfig)
	if err != nil {
		return err
	}
	defer ws.Close()

	var lokiResp LokiQueryTailResponse
	err = websocket.JSON.Receive(ws, &lokiResp)
	if err != nil {
		return err
	}

	if len(lokiResp.Streams) > 0 {
		now := time.Now()
		timeDiff := now.Sub(podInfo.StartTime)
		log.Printf("First log line for pod %s in namespace %s: (Time difference: %s)", podInfo.PodName, podInfo.Namespace, timeDiff)
		return nil
	}

	return fmt.Errorf("no logs found for pod %s", podInfo.PodName)
}

func main() {
	configFile := "config.yaml"

	configFileData, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer configFileData.Close()

	var config Config
	decoder := yaml.NewDecoder(configFileData)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	parseInputAndGetLokiLogs(config)
}

func parseInputAndGetLokiLogs(config Config) {
	var podStartTime time.Time
	var taskRunName string
	var targetNamespace string

	// Read JSON input from stdin
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var obj map[string]interface{}
		err := json.Unmarshal([]byte(line), &obj)
		if err != nil {
			log.Printf("Error unmarshaling JSON input: %v", err)
			continue
		}

		if val, ok := obj["podStartTime"].(string); ok {
			podStartTime, err = time.Parse(time.RFC3339, val)
			if err != nil {
				log.Printf("Error parsing podStartTime: %v", err)
				continue
			}
		}

		if val, ok := obj["taskRunName"].(string); ok {
			taskRunName = val
		}

		if val, ok := obj["targetNamespace"].(string); ok {
			targetNamespace = val
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading from stdin: %v", err)
		return
	}

	podInfo := PodInfo{
		Namespace: targetNamespace,
		PodName:   taskRunName,
		StartTime: podStartTime,
	}

	err := getTailLogsFromLoki(podInfo, config.LokiAddress, config.LokiWebSocketAddress, config)
	if err != nil {
		log.Printf("Error getting logs for pod %s in namespace %s: %v", podInfo.PodName, podInfo.Namespace, err)
	}
}
