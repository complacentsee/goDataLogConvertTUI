package main

import (
	"fmt"
	"log/slog"
	"math"
	"time"

	"complacentsee.com/goDataLogConvertTUI/helpers"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ScanningFilesPopupModel struct {
	Active                         bool
	TotalFiles                     int
	InitalizedFiles                int
	DATTagsLoadedFiles             int
	HistorianTagsLoadedFiles       int
	RecordLoadedFiles              int
	InitProgressBar                progress.Model
	InitPercentage                 float64
	DatTagsLoadedProgressBar       progress.Model
	DatTagsLoadedPercentage        float64
	HistorianTagsLoadedProgressBar progress.Model
	HistorianTagsLoadedPercentage  float64
	RecordsLoadedProgressBar       progress.Model
	RecordsLoadedPercentage        float64
}

func initialScanningPopupModel() ScanningFilesPopupModel {
	initpb := progress.New(progress.WithDefaultGradient())
	initpb.Width = 40

	dattagspb := progress.New(progress.WithDefaultGradient())
	dattagspb.Width = 40

	histtagspb := progress.New(progress.WithDefaultGradient())
	histtagspb.Width = 40

	recordspb := progress.New(progress.WithDefaultGradient())
	recordspb.Width = 40

	return ScanningFilesPopupModel{
		TotalFiles:                     1,
		InitalizedFiles:                0,
		DATTagsLoadedFiles:             0,
		HistorianTagsLoadedFiles:       0,
		RecordLoadedFiles:              0,
		Active:                         false,
		InitProgressBar:                initpb,
		InitPercentage:                 0.0,
		DatTagsLoadedProgressBar:       dattagspb,
		DatTagsLoadedPercentage:        0.0,
		HistorianTagsLoadedProgressBar: histtagspb,
		HistorianTagsLoadedPercentage:  0.0,
		RecordsLoadedProgressBar:       recordspb,
		RecordsLoadedPercentage:        0.0,
	}
}

type FileInititalCountMsg struct {
	FileCount int
}

func FileInititalCount(fileCount int) tea.Cmd {
	return func() tea.Msg {
		return FileInititalCountMsg{FileCount: fileCount}
	}
}

type FileScanCompletedMsg struct {
}

func FileScanCompleted() tea.Cmd {
	time.Sleep(1 * time.Second)
	return func() tea.Msg {
		return FileScanCompletedMsg{}
	}
}

func (m ScanningFilesPopupModel) View(width int, height int, background string) string {
	if !m.Active {
		return background
	}

	// Define the popup dimensions (smaller than the terminal)
	popupWidth := 50  // Adjust width as needed
	popupHeight := 12 // Adjust height based on the content

	// Create the border and content
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		Padding(1, 2).
		BorderForeground(lipgloss.Color("205"))

	initProgressView := m.InitProgressBar.ViewAs(m.InitPercentage)
	dattagsloadedProgressView := m.DatTagsLoadedProgressBar.ViewAs(m.DatTagsLoadedPercentage)
	histtagsloadedProgressView := m.HistorianTagsLoadedProgressBar.ViewAs(m.HistorianTagsLoadedPercentage)
	recordsloadedProgressView := m.RecordsLoadedProgressBar.ViewAs(m.RecordsLoadedPercentage)

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		fmt.Sprintf("%d files being processed, please wait...", m.TotalFiles),
		"Files scanned...",
		initProgressView,
		"Tags headers loaded...",
		dattagsloadedProgressView,
		"Historian tags validated...",
		histtagsloadedProgressView,
		"Record headers loaded...",
		recordsloadedProgressView,
	)

	forground := lipgloss.Place(
		popupWidth, popupHeight, // Popup width and height
		lipgloss.Center, lipgloss.Center, // Align the content in the center of the space
		borderStyle.Render(content),       // Rendered popup content with border
		lipgloss.WithWhitespaceChars(" "), // Fill the rest with spaces if needed
	)

	// Calculate x and y positions with explicit float arithmetic
	x := int(math.Round(float64(width)/2 - float64(popupWidth)*0.5))
	y := int(math.Round(float64(height)/2 - 2 - float64(popupHeight)*0.5))
	slog.Debug("Window popup:", "Pupup dims", fmt.Sprintf("width: %d, height: %d, x:%d, y:%d", width, height, x, y))

	// Return the overlay placement
	return helpers.PlaceOverlay(x, y, forground, background, false)

}
