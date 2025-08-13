package progress

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTracker(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewTracker(Options{
		Description:     "Test Progress",
		Total:           100,
		ShowElapsedTime: true,
		Width:           20,
		Writer:          &buf,
	})

	require.NotNil(t, tracker)
	assert.Equal(t, "Test Progress", tracker.description)
	assert.NotZero(t, tracker.startTime)
}

func TestTrackerUpdate(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewTracker(Options{
		Description: "Test Update",
		Total:       100,
		Writer:      &buf,
	})

	// Test update
	err := tracker.Update(50)
	assert.NoError(t, err)

	// Test add
	err = tracker.Add(25)
	assert.NoError(t, err)

	// Test finish
	err = tracker.Finish()
	assert.NoError(t, err)
}

func TestTrackerSetDescription(t *testing.T) {
	var buf bytes.Buffer
	tracker := NewTracker(Options{
		Description: "Initial",
		Total:       100,
		Writer:      &buf,
	})

	tracker.SetDescription("Updated Description")
	assert.Equal(t, "Updated Description", tracker.description)
}

func TestTrackerElapsedTime(t *testing.T) {
	tracker := NewTracker(Options{
		Description: "Test Time",
		Total:       100,
	})

	// Sleep for a short time
	time.Sleep(10 * time.Millisecond)

	elapsed := tracker.ElapsedTime()
	assert.GreaterOrEqual(t, elapsed.Milliseconds(), int64(10))
}

func TestMultiTracker(t *testing.T) {
	mt := NewMultiTracker()
	require.NotNil(t, mt)

	// Add trackers
	tracker1 := mt.Add("task1", Options{
		Description: "Task 1",
		Total:       100,
	})
	require.NotNil(t, tracker1)

	tracker2 := mt.Add("task2", Options{
		Description: "Task 2",
		Total:       200,
	})
	require.NotNil(t, tracker2)

	// Get tracker
	retrievedTracker, ok := mt.Get("task1")
	assert.True(t, ok)
	assert.Equal(t, tracker1, retrievedTracker)

	// Get non-existent tracker
	_, ok = mt.Get("nonexistent")
	assert.False(t, ok)

	// Remove tracker
	mt.Remove("task1")
	_, ok = mt.Get("task1")
	assert.False(t, ok)

	// Finish all
	mt.FinishAll() // Should not panic
}

func TestSpinner(t *testing.T) {
	spinner := NewSpinner("Test Spinner")
	require.NotNil(t, spinner)
	assert.Equal(t, "Test Spinner", spinner.text)

	// Test update text
	spinner.UpdateText("Updated Text")
	assert.Equal(t, "Updated Text", spinner.text)

	// Note: Start() and Stop() methods interact with terminal output
	// and use goroutines internally. Testing them here can cause race conditions
	// in test environments. These methods are tested implicitly through
	// actual usage in the CLI commands.
}

func TestFormatError(t *testing.T) {
	err := assert.AnError
	formatted := FormatError(err, "Test Context", "Check your configuration", "Try again later")

	// Check that formatted error contains expected parts
	assert.Contains(t, formatted, "Failed: Test Context")
	assert.Contains(t, formatted, err.Error())
	assert.Contains(t, formatted, "Suggestions:")
	assert.Contains(t, formatted, "Check your configuration")
	assert.Contains(t, formatted, "Try again later")
}

func TestFormatErrorNoSuggestions(t *testing.T) {
	err := assert.AnError
	formatted := FormatError(err, "Test Context")

	// Check that formatted error contains expected parts but no suggestions
	assert.Contains(t, formatted, "Failed: Test Context")
	assert.Contains(t, formatted, err.Error())
	assert.NotContains(t, formatted, "Suggestions:")
}
