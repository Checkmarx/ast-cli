package kicsshutdown

import (
	"sync"
	"testing"
)

func TestSetAndGetKicsContainerName(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
	}{
		{
			name:          "Set and get simple name",
			containerName: "test-container",
		},
		{
			name:          "Set and get name with uuid",
			containerName: "kics-scanner-12345678-1234-1234-1234-123456789012",
		},
		{
			name:          "Set and get empty name",
			containerName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetKicsContainerName(tt.containerName)
			got := GetKicsContainerName()
			if got != tt.containerName {
				t.Errorf("SetKicsContainerName(%s) -> GetKicsContainerName() = %s, want %s", tt.containerName, got, tt.containerName)
			}
		})
	}
}

func TestGetKicsContainerNameDefault(t *testing.T) {
	// Reset to empty state for this test
	SetKicsContainerName("")
	got := GetKicsContainerName()
	if got != "" {
		t.Errorf("GetKicsContainerName() without prior Set() = %s, want empty string", got)
	}
}

func TestKicsContainerNameOverwrite(t *testing.T) {
	SetKicsContainerName("first-container")
	first := GetKicsContainerName()
	if first != "first-container" {
		t.Errorf("First Set() failed: got %s, want first-container", first)
	}

	SetKicsContainerName("second-container")
	second := GetKicsContainerName()
	if second != "second-container" {
		t.Errorf("Second Set() failed: got %s, want second-container", second)
	}
}

func TestKicsContainerNameConcurrentAccess(t *testing.T) {
	SetKicsContainerName("")

	var wg sync.WaitGroup
	numGoroutines := 100
	testValue := "concurrent-test-container"

	// Launch multiple goroutines to test concurrent read/write
	for i := 0; i < numGoroutines/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			SetKicsContainerName(testValue)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			GetKicsContainerName()
		}()
	}

	wg.Wait()

	// Final value should be the test value (set by one of the goroutines)
	final := GetKicsContainerName()
	if final != testValue {
		t.Errorf("After concurrent access, got %s, want %s", final, testValue)
	}
}

func TestKicsContainerNameSequentialUpdates(t *testing.T) {
	names := []string{"container1", "container2", "container3", "container4", "container5"}

	for i, name := range names {
		SetKicsContainerName(name)
		got := GetKicsContainerName()
		if got != name {
			t.Errorf("Update %d: SetKicsContainerName(%s) -> GetKicsContainerName() = %s, want %s", i+1, name, got, name)
		}
	}

	// Final value should be the last one
	final := GetKicsContainerName()
	if final != names[len(names)-1] {
		t.Errorf("Final value = %s, want %s", final, names[len(names)-1])
	}
}

func TestKicsContainerNameRaceCondition(t *testing.T) {
	// This test is designed to detect race conditions when run with -race flag
	var wg sync.WaitGroup

	// Rapidly set and get the container name
	for i := 0; i < 50; i++ {
		wg.Add(2)

		go func(index int) {
			defer wg.Done()
			SetKicsContainerName("container-" + string(rune(index)))
		}(i)

		go func() {
			defer wg.Done()
			GetKicsContainerName()
		}()
	}

	wg.Wait()

	// Should complete without panicking or data races
	GetKicsContainerName()
}
