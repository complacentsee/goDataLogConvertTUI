package main

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/complacentsee/goDatalogConvert/LibDAT"
	"github.com/complacentsee/goDatalogConvert/LibPI"
)

type nullWriter struct{}

func (nw *nullWriter) Write(p []byte) (n int, err error) {
	// Discard the log message by returning the length of the input with no error
	return len(p), nil
}

func updateWithDATFileNameMsg(m model, msg DATFileNameMsg) (model, tea.Cmd) {
	m.UpdateViewDimentions()
	// validate column width
	requiredWidth := len(msg.fileName) + 1
	columns := m.filesTable.Columns()

	if columns[1].Width < requiredWidth {
		columns[1].Width = requiredWidth
		m.filesTable.SetColumns(columns)
	}

	// Add new row
	row := table.Row{"[X]", msg.fileName, "Pending", "", "", "", "", "", ""}
	m.rows = append(m.rows, row)
	m.filesTable.SetRows(m.rows)

	return m, LoadDATTagFile(m, msg.fileName)
}

// findRowByFileName searches for a row with the given file name.
func findRowByFileName(m model, fileName string) (int, table.Row, error) {
	for i, row := range m.rows {
		if len(row) > 0 && row[1] == fileName {
			return i, row, nil
		}
	}
	return -1, nil, fmt.Errorf("row with file name %s not found", fileName)
}

// updateRow updates the row at the specified index.
func updateRow(m model, index int, updatedRow table.Row) (model, error) {
	if index < 0 || index >= len(m.rows) {
		return m, fmt.Errorf("index out of bounds")
	}
	m.rows[index] = updatedRow
	m.filesTable.SetRows(m.rows)
	return m, nil
}

func updateWithDATFileHeaderMsg(m model, msg DATTagFileHeaderMsg) model {
	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		return m
	}
	updatedRow := table.Row{row[0], row[1], "Tags Loaded", msg.date, fmt.Sprintf("%d", msg.recordCound), row[5], row[6], row[7], row[8]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m
	}
	// Sort the rows by date after updating
	m.rows = sortRowsByDate(m.rows)
	m.filesTable.SetRows(m.rows)

	// update progress bar popup model
	m.sfmpu.InitalizedFiles++
	if m.sfmpu.TotalFiles > 0 {
		m.sfmpu.InitPercentage = float64(m.sfmpu.InitalizedFiles) / float64(m.sfmpu.TotalFiles)
	}
	return m
}

// parseDate parses a date string in "YYYY-MM-DD" format.
func parseDate(dateStr string) (time.Time, error) {
	return time.Parse("2006-01-02", dateStr)
}

// sortRowsByDate sorts the rows by date, pushing any without a date to the bottom.
func sortRowsByDate(rows []table.Row) []table.Row {
	sort.SliceStable(rows, func(i, j int) bool {
		dateI, errI := parseDate(rows[i][3])
		dateJ, errJ := parseDate(rows[j][3])
		if errI != nil && errJ != nil {
			return false // Both dates are invalid; keep their relative order
		}
		if errI != nil {
			return false // i has no valid date, so it should be after j
		}
		if errJ != nil {
			return true // j has no valid date, so i should come before j
		}
		return dateI.Before(dateJ)
	})
	return rows
}

type DATRecordStructure struct {
	TagRecords   []*LibDAT.DatTagRecord
	FloatRecords *[]*LibDAT.DatFloatRecord
	PointCache   *LibPI.PointLookup
	recordCount  int
}

func upsertDatTagFileRecord(m model, fileName string, records []*LibDAT.DatTagRecord) model {
	// Check if the entry exists
	if _, exists := m.datFileRecords[fileName]; !exists {
		// If it doesn't exist, create it
		m.datFileRecords[fileName] = DATRecordStructure{}
	}

	recordStruct := m.datFileRecords[fileName]
	recordStruct.TagRecords = records
	recordStruct.PointCache = LibPI.NewPointLookup()
	m.datFileRecords[fileName] = recordStruct

	// update progress bar popup model
	m.sfmpu.DATTagsLoadedFiles++
	if m.sfmpu.TotalFiles > 0 {
		m.sfmpu.DatTagsLoadedPercentage = float64(m.sfmpu.DATTagsLoadedFiles) / float64(m.sfmpu.TotalFiles)
	}
	return m
}

func updateDATFileRecord(m *model, msg LookupTagsOnHistorianMsg) error {
	// Check if the record exists and update or create it
	if record, exists := m.datFileRecords[msg.fileName]; exists {
		record.PointCache = msg.pointCache
		m.datFileRecords[msg.fileName] = record
	} else {
		m.datFileRecords[msg.fileName] = DATRecordStructure{
			PointCache: msg.pointCache,
		}
	}

	// Find the row by file name
	index, updatedRow, err := findRowByFileName(*m, msg.fileName)
	if err != nil {
		return err
	}

	// Update the row
	updatedRow[5] = fmt.Sprintf("%d", msg.validTags)
	updatedRow[2] = "Tags Valid"
	m.rows[index] = updatedRow
	m.filesTable.SetRows(m.rows)

	// update progress bar popup model
	m.sfmpu.HistorianTagsLoadedFiles++
	if m.sfmpu.TotalFiles > 0 {
		m.sfmpu.HistorianTagsLoadedPercentage = float64(m.sfmpu.HistorianTagsLoadedFiles) / float64(m.sfmpu.TotalFiles)
	}

	return nil
}

func updateWithDATFloatFileHeaderMsg(m model, msg DATFloatFileHeaderMsg) (model, tea.Cmd) {
	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		return m, nil
	}
	updatedRow := table.Row{row[0], row[1], row[2], row[3], row[4], row[5], fmt.Sprintf("%d", msg.recordCound), row[7], row[8]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m, nil
	}
	// Sort the rows by date after updating
	m.rows = sortRowsByDate(m.rows)
	m.filesTable.SetRows(m.rows)

	record := m.datFileRecords[msg.fileName]
	record.recordCount = int(msg.recordCound)
	m.datFileRecords[msg.fileName] = record

	// update progress bar popup model
	m.sfmpu.RecordLoadedFiles++
	if m.sfmpu.TotalFiles > 0 {
		m.sfmpu.RecordsLoadedPercentage = float64(m.sfmpu.RecordLoadedFiles) / float64(m.sfmpu.TotalFiles)
	}
	if m.sfmpu.RecordLoadedFiles == m.sfmpu.TotalFiles {
		return m, FileScanCompleted()
	}
	return m, nil
}

func updateWithDATFloatFileRecordsMsg(m model, msg DATTagFloatRecordMsg) model {
	// update progress bar
	m.processingStatus.datFilesProcessed++
	if m.processingStatus.processingCount > 0 {
		m.processingStatus.datFilesProcessedPBPercent = float64(m.processingStatus.datFilesProcessed) / float64(m.processingStatus.processingCount)
	}

	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		return m
	}

	record := m.datFileRecords[msg.fileName]
	record.FloatRecords = msg.records
	m.datFileRecords[msg.fileName] = record

	updatedRow := table.Row{row[0], row[1], "Recs loaded", row[3], row[4], row[5], row[6], fmt.Sprintf("%.2f sec", msg.duration.Seconds()), row[8]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m
	}

	return m
}

func updateWithUpdateStateToLoadingMsg(m model, msg UpdateStateToLoadingMsg) model {
	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		return m
	}

	updatedRow := table.Row{row[0], row[1], "Loading", row[3], row[4], row[5], row[6], row[7], row[8]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m
	}

	return m
}

// Helper function to process the next file
func processNextDatFile(m *model, first bool) (tea.Model, tea.Cmd) {
	if !m.connected {
		return m, SendStatus("Must be connected to server to process.")
	}

	if first {
		processCountTotal := 0
		for i := 0; i < len(m.rows); i++ {
			if m.rows[i][0] == "[X]" && m.rows[i][2] == "Tags Valid" {
				processCountTotal++
			}
		}
		m.InitializeProgressBars(processCountTotal)
	}

	for i := 0; i < len(m.rows); i++ {
		if m.rows[i][0] == "[X]" && m.rows[i][2] == "Tags Valid" {
			if m.recsLoadedCount < 3 {
				name := m.rows[i][1]
				m.rows[i][2] = "Processing"
				m.filesTable.SetRows(m.rows)
				m.recsLoadedCount++
				return m, tea.Batch(
					LoadDATFloatRecords(m, name, m.datFileRecords[name].recordCount),
					UpdateStateToLoading(name),
				)
			} else {
				return m, tea.Batch(
					RetriggerDATLoading(),
				)
			}
		}
	}
	if m.processed && !first {
		return m, SendStatus("Processing all DAT files completed successfully!")
	}

	return m, tea.Batch(SendStatus("No files in state ready for processing. Files should be Selected and marked \"Tags Valid\""),
		ResetProcessingFlag())
}

func processNextHistorianInsert(m *model) tea.Cmd {
	for i := 0; i < len(m.rows); i++ {
		if m.rows[i][0] == "[X]" && m.rows[i][2] == "Recs loaded" {
			name := m.rows[i][1]
			m.rows[i][2] = "Inserting"
			return InsertHistorianRecords(m, name)
		}
	}
	// TODO: Add completion logic here.
	return RetriggerHistorianInsert()
}

func updateWithHistorianInsertMsg(m model, msg HistorianInsertMsg) model {
	// update progress bar
	m.processingStatus.historianInserted++
	if m.processingStatus.processingCount > 0 {
		m.processingStatus.historianInsertedProcessedPBPercent = float64(m.processingStatus.historianInserted) / float64(m.processingStatus.processingCount)
	}

	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		updatedRow := table.Row{row[0], row[1], "Error Inserting", row[3], row[4], row[5], row[6], row[7], fmt.Sprintf("%.2f sec", msg.duration.Seconds())}
		m, _ = updateRow(m, index, updatedRow)
		return m
	}

	delete(m.datFileRecords, msg.fileName)
	m.recsLoadedCount--

	updatedRow := table.Row{row[0], row[1], "Completed", row[3], row[4], row[5], row[6], row[7], fmt.Sprintf("%.2f sec", msg.duration.Seconds())}
	m, _ = updateRow(m, index, updatedRow)

	return m
}

type RetriggerDATLoadingMsg struct {
}

func RetriggerDATLoading() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return RetriggerDATLoadingMsg{}
	}
}

type RetriggerHistorianInsertMsg struct {
}

func RetriggerHistorianInsert() tea.Cmd {
	return func() tea.Msg {
		time.Sleep(2 * time.Second)
		return RetriggerHistorianInsertMsg{}
	}
}

type StatusMsg struct {
	message string
}

func SendStatus(message string) tea.Cmd {
	return func() tea.Msg {
		return StatusMsg{message: message}
	}
}

type ResetProcessingFlagMsg struct {
}

func ResetProcessingFlag() tea.Cmd {
	return func() tea.Msg {
		return ResetProcessingFlagMsg{}
	}
}

func (m *model) UpdateViewDimentions() {
	newHeight := m.Height
	if newHeight < 1 {
		m.filesTable.SetHeight(1)
	} else {
		newHeight = newHeight - 2 //Remove top header for server status & bottom key menu
		if m.useTagMap {
			newHeight--
		}
		if m.processingStatus != nil {
			newHeight = newHeight - 2
		}
		if m.novfpu.Active {
			newHeight--
		}
		m.filesTable.SetHeight(newHeight)
	}

	if m.processingStatus != nil {
		m.processingStatus.datFilesProcessedPB.Width = m.Width/2 - 2
		m.processingStatus.historianInsertedProcessedPB.Width = m.Width/2 - 2
	}

}

type processingStatus struct {
	processingCount                     int
	datFilesProcessed                   int
	historianInserted                   int
	datFilesProcessedPB                 progress.Model
	datFilesProcessedPBPercent          float64
	historianInsertedProcessedPB        progress.Model
	historianInsertedProcessedPBPercent float64
}

func (m *model) InitializeProgressBars(totalProcessCount int) {
	datFilesProcessedPB := progress.New(progress.WithDefaultGradient())
	datFilesProcessedPB.Width = m.Width/2 - 2

	dattagspb := progress.New(progress.WithDefaultGradient())
	dattagspb.Width = m.Width/2 - 2

	m.processingStatus = &processingStatus{
		processingCount:                     totalProcessCount,
		datFilesProcessed:                   0,
		historianInserted:                   0,
		datFilesProcessedPB:                 datFilesProcessedPB,
		datFilesProcessedPBPercent:          0.0,
		historianInsertedProcessedPB:        dattagspb,
		historianInsertedProcessedPBPercent: 0.0,
	}

	m.UpdateViewDimentions()
}
