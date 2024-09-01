package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/complacentsee/goDatalogConvert/LibDAT"
)

type model struct {
	filesTable     table.Model
	selected       int
	rows           []table.Row
	connected      bool
	connecting     bool
	dirPath        string
	hostname       string
	processName    string
	tagMapCSV      string
	debugLevel     bool
	dr             *LibDAT.DatReader
	footerStatus   string
	datFileRecords map[string]DATRecordStructure
	tagMaps        map[string]string
	useTagMap      bool
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
		{Title: "Recs Written", Width: 12},
		{Title: "Duration", Width: 8},
	}

	rows := []table.Row{}

	filesTable := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true), // Highlight focused row
		table.WithHeight(15),    // Set height to fit the table
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
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			m.filesTable.MoveDown(1)
		case "k", "up":
			m.filesTable.MoveUp(1)
		case " ", "enter":
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
			if len(m.rows) > 0 {
				name := m.rows[0][1]
				return m, tea.Batch(LoadDATFloatRecords(&m, name, m.datFileRecords[name].recordCount),
					UpdateStateToLoading(name))
			}
		}
	case string:
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
		}
	case LookupTagsOnHistorianMsg:
		updateDATFileRecord(&m, msg)
		return m, LoadDATFloatFile(m, msg.fileName)
	case DATFloatFileHeaderMsg:
		m = updateWithDATFloatFileHeaderMsg(m, msg)
	case DATTagFloatRecordMsg:
		m = updateWithDATFloatFileRecordsMsg(m, msg)
	case UpdateStateToLoadingMsg:
		m = updateWithUpdateStateToLoadingMsg(m, msg)
	}

	return m, nil
}

func (m model) View() string {
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

func main() {
	// Define the command-line flags
	dirPath := flag.String("path", ".", "Path to the directory containing DAT files")
	host := flag.String("host", "localhost", "Hostname of PI server")
	processName := flag.String("processName", "dat2fth", "Process name")
	tagMapCSV := flag.String("tagMapCSV", "", "Path to the CSV file containing the tag map.")
	debugLevel := flag.Bool("debug", false, "Enable debug logging")

	// Set up slog to use a null writer
	nullHandler := slog.NewTextHandler(&nullWriter{}, nil)
	logger := slog.New(nullHandler)
	slog.SetDefault(logger)

	// Parse the flags
	flag.Parse()

	// Initialize the Bubble Tea program with the flags
	p := tea.NewProgram(initialModel(*dirPath, *host, *processName, *tagMapCSV, *debugLevel))

	// Run the Bubble Tea program
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
