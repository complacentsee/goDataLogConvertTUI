package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/complacentsee/goDatalogConvert/LibDAT"
)

type model struct {
	Width, Height    int
	filesTable       table.Model
	selected         int
	rows             []table.Row
	connected        bool
	connecting       bool
	dirPath          string
	hostname         string
	processName      string
	tagMapCSV        string
	debugLevel       bool
	dr               *LibDAT.DatReader
	footerStatus     string
	datFileRecords   map[string]DATRecordStructure
	tagMaps          map[string]string
	useTagMap        bool
	processed        bool
	firstDatReturned bool
	recsLoadedCount  int
	sfmpu            ScanningFilesPopupModel
}

func initialModel(dirPath, host, processName, tagMapCSV string, debugLevel bool) model {
	footerStatus := ""

	dr, err := LibDAT.NewDatReader(dirPath)
	if err != nil {
		footerStatus = fmt.Sprintf("Unable to find valid failes in directory: %s", dirPath)
	}

	// Initialize the table with columns and rows
	columns := []table.Column{
		{Title: "", Width: 3},
		{Title: "File Name", Width: 45},
		{Title: "State", Width: 12},
		{Title: "Date", Width: 10},
		{Title: "Dat Tags", Width: 8},
		{Title: "Hist Tags", Width: 9},
		{Title: "Records", Width: 7},
		{Title: "Duration", Width: 8},
		{Title: "Duration", Width: 8},
	}

	rows := []table.Row{}

	filesTable := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true), // Highlight focused row
		table.WithHeight(10),    // Set height to fit the table
	)

	return model{
		filesTable:     filesTable,
		rows:           rows,
		selected:       0,
		dirPath:        dirPath,
		hostname:       host,
		processName:    processName,
		tagMapCSV:      tagMapCSV,
		debugLevel:     debugLevel,
		connecting:     true,
		dr:             dr,
		footerStatus:   footerStatus,
		datFileRecords: make(map[string]DATRecordStructure),
		tagMaps:        make(map[string]string),
		useTagMap:      false,
		processed:      false,
		sfmpu:          initialScanningPopupModel(),
	}
}

func (m model) Init() tea.Cmd {
	// Combine multiple commands using tea.Batch
	return tea.Batch(
		SetProcessName(m.processName),
		PiConnectToServer(m.hostname),
		loadDirectory(m),
		LoadCSVMapping(m),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	slog.Debug("Update model called", "Type", fmt.Sprintf("%T", msg), "Tea.Msg", msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.filesTable.MoveDown(1)
		case "k", "up":
			m.filesTable.MoveUp(1)
		case "pgdown", "f", "ctrl+d": // Page Down
			m.filesTable.MoveDown(m.filesTable.Height()) // Move down by the height of the table
		case "pgup", "b", "ctrl+u": // Page Up
			m.filesTable.MoveUp(m.filesTable.Height()) // Move up by the height of the table

		case " ", "enter":
			if m.processed {
				return m, nil
			}
			selectedRow := m.filesTable.Cursor()
			if selectedRow >= 0 && selectedRow < len(m.rows) {
				if m.rows[selectedRow][0] == "[ ]" {
					m.rows[selectedRow][0] = "[X]"
				} else {
					m.rows[selectedRow][0] = "[ ]"
				}
			}
			m.filesTable.SetRows(m.rows)
		case "a":
			for i := 0; i < len(m.rows); i++ {
				m.rows[i][0] = "[X]"
			}
			m.filesTable.SetRows(m.rows)
		case "n":
			for i := 0; i < len(m.rows); i++ {
				m.rows[i][0] = "[ ]"
			}
			m.filesTable.SetRows(m.rows)
		case "p": // Process selected file
			if !m.processed {
				m.processed = true
				return processNextDatFile(&m, true)
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.UpdateTableHeight()

	case tea.MouseMsg:
		// Handle mouse scroll
		if msg.Type == tea.MouseWheelUp {
			m.filesTable.MoveUp(1)
		} else if msg.Type == tea.MouseWheelDown {
			m.filesTable.MoveDown(1)
		}

	case PiServerProcessNameMsg:
		m.processName = msg.processName
	case PiServerConnectMsg:
		m.connected = msg.connected
		m.hostname = msg.hostname
		m.connecting = false
	case DATFileNameMsg:
		return updateWithDATFileNameMsg(m, msg)
	case DATTagFileHeaderMsg:
		m = updateWithDATFileHeaderMsg(m, msg)
		return m, tea.Batch(LoadDATTagRecords(m, msg.fileName, int(msg.recordCound)))
	case DATTagRecordMsg:
		m = upsertDatTagFileRecord(m, msg.fileName, msg.records)
		return m, LookupTagsOnHistorian(m, msg.fileName)
	case CSVMapping:
		if msg.err == "" {
			m.tagMaps = msg.mapping
			m.useTagMap = true
		} else {
			m.footerStatus = msg.err
		}
	case LookupTagsOnHistorianMsg:
		updateDATFileRecord(&m, msg)
		return m, LoadDATFloatFile(m, msg.fileName)
	case DATFloatFileHeaderMsg:
		return updateWithDATFloatFileHeaderMsg(m, msg)
	case DATTagFloatRecordMsg:
		m = updateWithDATFloatFileRecordsMsg(m, msg)
		if !m.firstDatReturned {
			m.firstDatReturned = true
			histCMD := processNextHistorianInsert(&m)
			m, cmd := processNextDatFile(&m, false)
			return m, tea.Batch(cmd, histCMD)
		}
		m.firstDatReturned = true
		return processNextDatFile(&m, false)
	case UpdateStateToLoadingMsg:
		m = updateWithUpdateStateToLoadingMsg(m, msg)
	case HistorianInsertMsg:
		m = updateWithHistorianInsertMsg(m, msg)
		return m, processNextHistorianInsert(&m)
	case RetriggerDATLoadingMsg:
		return processNextDatFile(&m, false)
	case RetriggerHistorianInsertMsg:
		return m, processNextHistorianInsert(&m)
	case StatusMsg:
		m.footerStatus = msg.message
	case ResetProcessingFlagMsg:
		m.processed = false

	case FileInititalCountMsg:
		m.sfmpu.TotalFiles = msg.FileCount
		m.sfmpu.Active = true
	case FileScanCompletedMsg:
		m.sfmpu.Active = false
	}

	return m, nil
}

func (m model) View() string {
	s := ""
	s += m.ViewMainModel()

	if m.sfmpu.Active {
		s = m.sfmpu.View(m.Width, m.Height, s)
	}

	return s
}

func main() {
	// Define the command-line flags
	dirPath := flag.String("path", ".", "Path to the directory containing DAT files")
	host := flag.String("host", "localhost", "Hostname of PI server")
	processName := flag.String("processName", "dat2fth", "Process name")
	tagMapCSV := flag.String("tagMapCSV", "", "Path to the CSV file containing the tag map.")
	debugLevel := flag.Bool("debug", false, "Enable debug logging")

	// Parse the flags
	flag.Parse()

	var logHandler *slog.TextHandler
	if debugLevel != nil && *debugLevel {
		// Handle debug logging to a text file
		logFile, err := os.OpenFile("applog.txt", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			slog.Error("Failed to open log file", "error", err)
			return
		}
		defer logFile.Close()
		logHandler = slog.NewTextHandler(logFile, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})
	} else {
		logHandler = slog.NewTextHandler(&nullWriter{}, nil)
	}
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// Initialize the Bubble Tea program with the flags
	p := tea.NewProgram(initialModel(*dirPath, *host, *processName, *tagMapCSV, *debugLevel), tea.WithAltScreen())

	// Run the Bubble Tea program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
