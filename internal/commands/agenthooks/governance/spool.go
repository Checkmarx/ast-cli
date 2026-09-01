package governance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	spoolBatchSize  = 50
	flushTimeout    = 2 * time.Second
)

// Write atomically appends an ActivityEvent to the audit spool.
// The event is first written to a .tmp file, then renamed, ensuring no partial files
// are ever visible to the flusher. Spool write failures are non-fatal — the verdict
// has already been returned to the agent.
func Write(ev ActivityEvent) {
	dir := spoolDir()
	ensureDir(dir)

	data, err := json.Marshal(ev)
	if err != nil {
		log.Printf("governance: spool marshal error: %v", err)
		return
	}

	tmp := filepath.Join(dir, "evt_"+ev.EventID+".tmp")
	dest := filepath.Join(dir, "evt_"+ev.EventID+".json")

	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		log.Printf("governance: spool write error: %v", err)
		return
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		log.Printf("governance: spool rename error: %v", err)
	}
}

// FlushOnce reads all .json files from the spool, sends them to the backend in batches,
// and removes successfully posted events. 4xx responses move events to the dead-letter
// directory; 5xx and network errors leave events in place for the next flush attempt.
// timeout bounds the total time spent; zero means no deadline.
func FlushOnce(serverURL, token string, timeout time.Duration) {
	if serverURL == "" {
		return
	}
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}

	pattern := filepath.Join(spoolDir(), "*.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return
	}

	for i := 0; i < len(files); i += spoolBatchSize {
		if !deadline.IsZero() && time.Now().After(deadline) {
			return
		}
		end := i + spoolBatchSize
		if end > len(files) {
			end = len(files)
		}
		if err := flushBatch(serverURL, token, files[i:end]); err != nil {
			log.Printf("governance: spool flush error: %v", err)
		}
	}
}

// flushBatch reads the given spool files, POSTs them to the backend, and
// removes files on success. Moves to dead/ on permanent 4xx failure.
func flushBatch(serverURL, token string, files []string) error {
	events := make([]ActivityEvent, 0, len(files))
	fileMap := make(map[string]ActivityEvent, len(files))

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var ev ActivityEvent
		if err := json.Unmarshal(data, &ev); err != nil {
			moveToDeadLetter(f)
			continue
		}
		events = append(events, ev)
		fileMap[f] = ev
	}
	if len(events) == 0 {
		return nil
	}

	payload := map[string]any{"events": events}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("spool: marshal batch: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, serverURL+"/api/v1/audit/events", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err // transient — leave files, retry next time
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		for f := range fileMap {
			_ = os.Remove(f)
		}
	case resp.StatusCode >= 400 && resp.StatusCode < 500:
		// Permanent failure — move to dead-letter queue, stop retrying
		for f := range fileMap {
			moveToDeadLetter(f)
		}
		return fmt.Errorf("spool: backend rejected batch with %d", resp.StatusCode)
	default:
		return fmt.Errorf("spool: backend returned %d", resp.StatusCode)
	}
	return nil
}

// moveToDeadLetter moves a spool file to audit-queue/dead/ so the main queue
// stays clean while preserving the rejected payload for investigation.
func moveToDeadLetter(src string) {
	deadDir := filepath.Join(spoolDir(), "dead")
	ensureDir(deadDir)
	dest := filepath.Join(deadDir, filepath.Base(src))
	if err := os.Rename(src, dest); err != nil {
		_ = os.Remove(src)
	}
}

