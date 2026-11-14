package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFileFlowServiceWakeNamesDurableQueueAndEvent(t *testing.T) {
	now := time.Date(2026, 9, 23, 5, 0, 0, 0, time.UTC)
	data, err := buildFileFlowServiceWake(`C:\Users\x\fileflow.jsonl`, "metadata:journal:abc", now)
	if err != nil {
		t.Fatal(err)
	}
	var wake FileFlowServiceWake
	if err := json.Unmarshal(data, &wake); err != nil {
		t.Fatal(err)
	}
	if wake.Action != "process_fileflow_queue" || wake.EventID != "metadata:journal:abc" || wake.QueuePath == "" {
		t.Fatalf("unexpected wake %#v", wake)
	}
}
