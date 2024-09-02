package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m model) ViewMainModel() string {
	var statusColor lipgloss.Color
	statusMessage := ""

	if m.connecting {
		statusColor = lipgloss.Color("3")
		statusMessage = fmt.Sprintf("Connecting to Server: %s, with process name: %s", m.hostname, m.processName)
	} else {
		if m.connected {
			statusColor = lipgloss.Color("82")
			statusMessage = fmt.Sprintf("Connected to Server: %s, with process name: %s", m.hostname, m.processName)
		} else {
			statusColor = lipgloss.Color("1")
			statusMessage = fmt.Sprintf("Unable to connect to server: %s, with process name: %s", m.hostname, m.processName)
		}
	}

	statusStyle := lipgloss.NewStyle().Foreground(statusColor).Render

	// Status bar
	s := fmt.Sprintf("Server status: %s\n", statusStyle(statusMessage))
	if m.useTagMap {
		s += fmt.Sprintf("Using tag map file: %s\n", m.tagMapCSV)
	}

	// Render the table
	s += m.filesTable.View()
	s += "\n"
	if m.footerStatus != "" {
		s += fmt.Sprintf("Status: %s\n", m.footerStatus)
	}
	s += "[q] Quit  [j] Down  [k] Up  [Space/Enter] Toggle Select [a] Select All  [n] Deselect All  [p] Process All\n"
	return s
}
