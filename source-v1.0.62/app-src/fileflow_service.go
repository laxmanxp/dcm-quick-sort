package main

import (
	"encoding/json"
	"strings"
	"time"
)

const fileFlowServicePipeName = `\\.\pipe\flowverse`

type FileFlowServiceWake struct {
	Version   int       `json:"version"`
	Action    string    `json:"action"`
	QueuePath string    `json:"queue_path"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`
}

func buildFileFlowServiceWake(queuePath, eventID string, now time.Time) ([]byte, error) {
	wake := FileFlowServiceWake{Version: 1, Action: "process_fileflow_queue", QueuePath: strings.TrimSpace(queuePath), EventID: strings.TrimSpace(eventID), CreatedAt: now}
	data, err := json.Marshal(wake)
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func notifyFileFlowService(queuePath, eventID string, now time.Time) error {
	payload, err := buildFileFlowServiceWake(queuePath, eventID, now)
	if err != nil {
		return err
	}
	return sendFlowverseServiceWake(payload)
}
