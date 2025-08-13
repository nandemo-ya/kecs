// Package progress provides enhanced terminal progress visualization for KECS operations
package progress

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/pterm/pterm"
	"github.com/schollz/progressbar/v3"
)

// Tracker provides progress tracking for a single operation
type Tracker struct {
	bar         *progressbar.ProgressBar
	description string
	startTime   time.Time
	mu          sync.Mutex
}

// MultiTracker manages multiple progress bars for parallel operations
type MultiTracker struct {
	trackers map[string]*Tracker
	mu       sync.RWMutex
	output   io.Writer
}

// Options configures progress bar appearance and behavior
type Options struct {
	Description     string
	Total           int64
	ShowElapsedTime bool
	ShowETA         bool
	Width           int
	Writer          io.Writer
}

// NewTracker creates a new progress tracker for a single operation
func NewTracker(opts Options) *Tracker {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}
	if opts.Width == 0 {
		opts.Width = 40
	}

	barOpts := []progressbar.Option{
		progressbar.OptionSetDescription(opts.Description),
		progressbar.OptionSetWriter(opts.Writer),
		progressbar.OptionSetWidth(opts.Width),
		progressbar.OptionThrottle(100 * time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionClearOnFinish(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "█",
			SaucerHead:    "█",
			SaucerPadding: "░",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	}

	if opts.ShowElapsedTime {
		barOpts = append(barOpts, progressbar.OptionShowElapsedTimeOnFinish())
	}
	if opts.ShowETA {
		barOpts = append(barOpts, progressbar.OptionShowIts())
	}

	bar := progressbar.NewOptions64(opts.Total, barOpts...)

	return &Tracker{
		bar:         bar,
		description: opts.Description,
		startTime:   time.Now(),
	}
}

// Update updates the progress bar with the current value
func (t *Tracker) Update(value int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.bar.Set64(value)
}

// Add increments the progress bar by the given amount
func (t *Tracker) Add(delta int64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.bar.Add64(delta)
}

// SetDescription updates the description text
func (t *Tracker) SetDescription(desc string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.description = desc
	t.bar.Describe(desc)
}

// Finish completes the progress bar
func (t *Tracker) Finish() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.bar.Finish()
}

// FinishWithMessage completes the progress bar with a custom message
func (t *Tracker) FinishWithMessage(msg string) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.bar.Describe(msg)
	return t.bar.Finish()
}

// ElapsedTime returns the time elapsed since the tracker was created
func (t *Tracker) ElapsedTime() time.Duration {
	return time.Since(t.startTime)
}

// NewMultiTracker creates a tracker for managing multiple parallel progress bars
func NewMultiTracker() *MultiTracker {
	return &MultiTracker{
		trackers: make(map[string]*Tracker),
		output:   os.Stdout,
	}
}

// Add adds a new progress tracker with the given ID
func (mt *MultiTracker) Add(id string, opts Options) *Tracker {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	tracker := NewTracker(opts)
	mt.trackers[id] = tracker
	return tracker
}

// Get returns the tracker with the given ID
func (mt *MultiTracker) Get(id string) (*Tracker, bool) {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	tracker, ok := mt.trackers[id]
	return tracker, ok
}

// Remove removes the tracker with the given ID
func (mt *MultiTracker) Remove(id string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	delete(mt.trackers, id)
}

// FinishAll finishes all active trackers
func (mt *MultiTracker) FinishAll() {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	for _, tracker := range mt.trackers {
		tracker.Finish()
	}
}

// Spinner creates a simple spinner for indeterminate progress
type Spinner struct {
	spinner *pterm.SpinnerPrinter
	text    string
}

// NewSpinner creates a new spinner with the given text
func NewSpinner(text string) *Spinner {
	spinner := pterm.DefaultSpinner.WithText(text)
	return &Spinner{
		spinner: spinner,
		text:    text,
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.spinner, _ = s.spinner.Start()
}

// UpdateText updates the spinner text
func (s *Spinner) UpdateText(text string) {
	s.text = text
	s.spinner.UpdateText(text)
}

// Success stops the spinner with a success message
func (s *Spinner) Success(msg string) {
	s.spinner.Success(msg)
}

// Fail stops the spinner with a failure message
func (s *Spinner) Fail(msg string) {
	s.spinner.Fail(msg)
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.spinner.Stop()
}

// SectionHeader prints a formatted section header
func SectionHeader(title string) {
	pterm.DefaultSection.Println(title)
}

// Success prints a success message
func Success(format string, a ...interface{}) {
	pterm.Success.Printfln(format, a...)
}

// Error prints an error message
func Error(format string, a ...interface{}) {
	pterm.Error.Printfln(format, a...)
}

// Info prints an info message
func Info(format string, a ...interface{}) {
	pterm.Info.Printfln(format, a...)
}

// Warning prints a warning message
func Warning(format string, a ...interface{}) {
	pterm.Warning.Printfln(format, a...)
}

// FormatError formats an error with additional context and suggestions
func FormatError(err error, context string, suggestions ...string) string {
	var result string
	
	// Error header
	result += pterm.Error.Sprint("Failed: " + context) + "\n\n"
	
	// Error details
	result += pterm.DefaultBox.Sprint(fmt.Sprintf("Error: %v", err)) + "\n"
	
	// Suggestions if provided
	if len(suggestions) > 0 {
		result += "\n" + pterm.Info.Sprint("Suggestions:") + "\n"
		for _, suggestion := range suggestions {
			result += fmt.Sprintf("  • %s\n", suggestion)
		}
	}
	
	return result
}