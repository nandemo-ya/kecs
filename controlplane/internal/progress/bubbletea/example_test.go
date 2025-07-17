package bubbletea_test

import (
	"fmt"
	"time"

	"github.com/nandemo-ya/kecs/controlplane/internal/progress"
	"github.com/nandemo-ya/kecs/controlplane/internal/progress/bubbletea"
)

func ExampleRunWithProgress() {
	// Example of using the Bubble Tea progress display
	err := bubbletea.RunWithProgress("Example Task", func(prog *bubbletea.Program) error {
		// Add tasks
		prog.AddTask("download", "Downloading files", 100)
		prog.AddTask("process", "Processing data", 100)
		prog.AddTask("upload", "Uploading results", 100)
		
		// Simulate download
		for i := 0; i <= 100; i += 10 {
			prog.UpdateTask("download", float64(i), fmt.Sprintf("Downloading... %d%%", i))
			time.Sleep(100 * time.Millisecond)
		}
		prog.CompleteTask("download")
		
		// Simulate processing
		for i := 0; i <= 100; i += 5 {
			prog.UpdateTask("process", float64(i), fmt.Sprintf("Processing batch %d/20", i/5))
			prog.Log("INFO", "Processing item %d", i)
			time.Sleep(50 * time.Millisecond)
		}
		prog.CompleteTask("process")
		
		// Simulate upload
		for i := 0; i <= 100; i += 20 {
			prog.UpdateTask("upload", float64(i), fmt.Sprintf("Uploading chunk %d/5", i/20+1))
			time.Sleep(200 * time.Millisecond)
		}
		prog.CompleteTask("upload")
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func ExampleAdapter() {
	// Example of using the adapter that mimics ParallelTracker
	adapter := bubbletea.NewAdapter("Deployment Progress")
	
	// Start the display
	if err := adapter.Start(); err != nil {
		panic(err)
	}
	defer adapter.Stop()
	
	// Add tasks
	adapter.AddTask("backend", "Backend Service", 100)
	adapter.AddTask("frontend", "Frontend Service", 100)
	
	// Update progress
	adapter.StartTask("backend")
	for i := 0; i <= 100; i += 10 {
		adapter.UpdateTask("backend", i, fmt.Sprintf("Deploying... %d%%", i))
		adapter.Log(progress.LogLevelInfo, "Backend progress: %d%%", i)
		time.Sleep(100 * time.Millisecond)
	}
	adapter.CompleteTask("backend")
	
	// Simulate frontend deployment
	adapter.StartTask("frontend") 
	for i := 0; i <= 100; i += 5 {
		adapter.UpdateTask("frontend", i, fmt.Sprintf("Building assets... %d%%", i))
		time.Sleep(50 * time.Millisecond)
	}
	adapter.CompleteTask("frontend")
}