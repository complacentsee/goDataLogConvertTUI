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
	if m.processed && m.processingStatus != nil {
		s += m.ViewProcessingProgressBar()
	}
	s += "[q] Quit  [j] Down  [k] Up  [Space/Enter] Toggle Select [a] Select All  [n] Deselect All  [p] Process All"
	return s
}

func (m model) ViewProcessingProgressBar() string {
	if m.processingStatus != nil {

		datProgressView := m.processingStatus.datFilesProcessedPB.ViewAs(m.processingStatus.datFilesProcessedPBPercent)
		datFilesProcessed := lipgloss.JoinVertical(
			lipgloss.Left,
			fmt.Sprintf("%d dat file records loaded out of %d.", m.processingStatus.datFilesProcessed, m.processingStatus.processingCount),
			datProgressView,
		)

		historianInsertProgressView := m.processingStatus.historianInsertedProcessedPB.ViewAs(m.processingStatus.historianInsertedProcessedPBPercent)
		historianInsertsProcessed := lipgloss.JoinVertical(
			lipgloss.Left,
			fmt.Sprintf("%d dat files inserted into historian out of %d.", m.processingStatus.historianInserted, m.processingStatus.processingCount),
			historianInsertProgressView,
		)

		progressBars := lipgloss.JoinHorizontal(
			lipgloss.Top,
			datFilesProcessed,
			historianInsertsProcessed,
		)
		progressBars += "\n"
		return progressBars
	}

	return ""
}
