package main

import (
	"fmt"
	"sort"
	"time"

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
	// Calculate the required width for the file name column
	requiredWidth := len(msg.fileName) + 2 // Adding some padding for readability

	// Get the current columns
	columns := m.filesTable.Columns()

	// Check if the current width is less than the required width
	if columns[1].Width < requiredWidth {
		// Update the width of the first column
		columns[1].Width = requiredWidth
		m.filesTable.SetColumns(columns)
	}

	// Add the new row
	row := table.Row{"[X]", msg.fileName, "Pending", "", "", "", "", "", "", ""}
	m.rows = append(m.rows, row)
	m.filesTable.SetRows(m.rows)

	// Load the DAT tag file (if necessary)
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
	updatedRow := table.Row{row[0], row[1], "Tags Loaded", msg.date, fmt.Sprintf("%d", msg.recordCound), row[5], row[6], row[7], row[8], row[9]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m
	}
	// Sort the rows by date after updating
	m.rows = sortRowsByDate(m.rows)
	m.filesTable.SetRows(m.rows)

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

	return nil
}

func updateWithDATFloatFileHeaderMsg(m model, msg DATFloatFileHeaderMsg) model {
	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		return m
	}
	updatedRow := table.Row{row[0], row[1], row[2], row[3], row[4], row[5], fmt.Sprintf("%d", msg.recordCound), row[7], row[8], row[9]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m
	}
	// Sort the rows by date after updating
	m.rows = sortRowsByDate(m.rows)
	m.filesTable.SetRows(m.rows)

	record := m.datFileRecords[msg.fileName]
	record.recordCount = int(msg.recordCound)
	m.datFileRecords[msg.fileName] = record

	return m
}

func updateWithDATFloatFileRecordsMsg(m model, msg DATTagFloatRecordMsg) model {
	index, row, err := findRowByFileName(m, msg.fileName)
	if err != nil {
		return m
	}

	record := m.datFileRecords[msg.fileName]
	record.FloatRecords = msg.records
	m.datFileRecords[msg.fileName] = record

	updatedRow := table.Row{row[0], row[1], "Recs loaded", row[3], row[4], row[5], row[6], fmt.Sprintf("%.2f sec", msg.duration.Seconds()), row[8], row[9]}
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

	updatedRow := table.Row{row[0], row[1], "Loading", row[3], row[4], row[5], row[6], row[7], row[8], row[9]}
	m, err = updateRow(m, index, updatedRow)
	if err != nil {
		fmt.Println("Error updating row:", err)
		return m
	}

	return m
}
