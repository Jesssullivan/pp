package app

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TickCmd returns a bubbletea Cmd that sends a TickEvent after the given
// duration. This drives the periodic UI refresh cycle.
func TickCmd(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return TickEvent{Time: t}
	})
}

// DataFetchCmd returns a Cmd that runs fetchFn in a goroutine and delivers
// the result as a DataUpdateEvent. If fetchFn returns an error, the event's
// Err field is set and Data is nil.
//
// Usage:
//
//	cmd := DataFetchCmd("tailscale", func() (interface{}, error) {
//	    return collector.Fetch()
//	})
func DataFetchCmd(source string, fetchFn func() (interface{}, error)) tea.Cmd {
	return func() tea.Msg {
		data, err := fetchFn()
		return DataUpdateEvent{
			Source:    source,
			Data:     data,
			Err:      err,
			Timestamp: time.Now(),
		}
	}
}
