package main

import (
	"fmt"
	"log/slog"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/complacentsee/goDatalogConvert/LibDAT"
	"github.com/complacentsee/goDatalogConvert/LibFTH"
	"github.com/complacentsee/goDatalogConvert/LibPI"
	"github.com/complacentsee/goDatalogConvert/LibUtil"
)

type PiServerProcessNameMsg struct {
	processName string
}

func SetProcessName(processName string) tea.Cmd {
	return func() tea.Msg {
		LibFTH.SetProcessName(processName)
		return PiServerProcessNameMsg{processName: processName}
	}
}

type PiServerConnectMsg struct {
	connected bool
	hostname  string
	err       string
}

func PiConnectToServer(hostname string) tea.Cmd {
	return func() tea.Msg {
		err := LibFTH.Connect(hostname)
		if err != nil {
			slog.Error(err.Error())
			return PiServerConnectMsg{connected: false, hostname: hostname, err: err.Error()}
		}

		return PiServerConnectMsg{connected: true, hostname: hostname, err: ""}
	}
}

type DATFileNameMsg struct {
	index    int
	fileName string
}

func loadDirectory(m model) tea.Cmd {
	var cmds []tea.Cmd

	// Iterate through the files and create a command for each
	filecount := 0
	for i, floatfileName := range m.dr.GetFloatFiles() {
		// Append the command to the list
		cmds = append(cmds, func(index int, fileName string) tea.Cmd {
			return func() tea.Msg {
				return DATFileNameMsg{index: index, fileName: fileName}
			}
		}(i, floatfileName))
		filecount++
	}

	// Initialize the popup.
	cmds = append(cmds, FileInititalCount(filecount))

	// Return all commands as a batch
	return tea.Batch(cmds...)
}

type DATTagFileHeaderMsg struct {
	fileName    string
	recordCound int32
	date        string
}

func LoadDATTagFile(m model, file string) tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, func() tea.Msg {
		records, date, err := m.dr.ReadTagFileHeader(file)
		if err != nil {
			return nil
		}
		return DATTagFileHeaderMsg{fileName: file, recordCound: *records, date: *date}
	})

	return tea.Batch(cmds...)
}

type DATTagRecordMsg struct {
	fileName string
	records  []*LibDAT.DatTagRecord
}

func LoadDATTagRecords(m model, file string, count int) tea.Cmd {
	return func() tea.Msg {
		records, err := m.dr.ReadTagRecordsFile(file, count)
		if err != nil {
			return nil
		}
		return DATTagRecordMsg{fileName: file, records: records}
	}
}

type CSVMapping struct {
	mapping map[string]string
	err     string
}

func LoadCSVMapping(m model) tea.Cmd {
	return func() tea.Msg {
		if m.tagMapCSV == "" {
			return nil
		}

		err := LibUtil.LoadTagMapCSV(m.tagMapCSV, m.tagMaps)
		if err != nil {
			slog.Error(fmt.Sprintf("Failed to load tag map CSV: %v", err))
			return CSVMapping{mapping: m.tagMaps, err: fmt.Sprintf("Failed to load tag map CSV: %v", err)}
		}
		if len(m.tagMaps) < 1 {
			return CSVMapping{mapping: m.tagMaps, err: "Tag mapping file was provided but had no entries."}
		}
		return CSVMapping{mapping: m.tagMaps, err: ""}
	}
}

type LookupTagsOnHistorianMsg struct {
	fileName   string
	pointCache *LibPI.PointLookup
	validTags  int
}

func LookupTagsOnHistorian(m model, filename string) tea.Cmd {
	return func() tea.Msg {
		if tagRecords, exists := m.datFileRecords[filename]; exists {
			count := 0
			for _, tag := range tagRecords.TagRecords {
				tagName := tag.Name
				if m.useTagMap {
					var exists bool
					tagName, exists = m.tagMaps[tag.Name]
					if !exists {
						continue
					}
				}

				LibDAT.PrintTagRecord(tag)
				_, exists := tagRecords.PointCache.GetPointByDataLogName(tag.Name)
				if exists {
					continue
				}
				pointC := LibFTH.AddToPIPointCache(tag.Name, tag.ID, 0, tagName)
				if !pointC.Process {
					continue
				}
				count++
				tagRecords.PointCache.AddPoint(pointC)
			}
			return LookupTagsOnHistorianMsg{fileName: filename, validTags: count, pointCache: tagRecords.PointCache}
		}
		return nil
	}
}

type DATFloatFileHeaderMsg struct {
	fileName    string
	recordCound int32
}

func LoadDATFloatFile(m model, file string) tea.Cmd {
	return func() tea.Msg {
		records, err := m.dr.ReadFloatFileHeader(file)
		if err != nil {
			return nil
		}
		return DATFloatFileHeaderMsg{fileName: file, recordCound: *records}
	}
}

type DATTagFloatRecordMsg struct {
	fileName string
	err      string
	duration time.Duration
	records  *[]*LibDAT.DatFloatRecord
}

func LoadDATFloatRecords(m *model, fileName string, recordCount int) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		records, err := m.dr.ReadFloatFileRecords(fileName, int32(recordCount))
		if err != nil {
			return DATTagFloatRecordMsg{fileName: fileName, err: err.Error()}
		}

		duration := time.Since(start)
		return DATTagFloatRecordMsg{fileName: fileName, records: &records, duration: duration, err: ""}
	}
}

type UpdateStateToLoadingMsg struct {
	fileName string
}

func UpdateStateToLoading(fileName string) tea.Cmd {
	return func() tea.Msg { return UpdateStateToLoadingMsg{fileName: fileName} }
}

type HistorianInsertMsg struct {
	fileName string
	err      string
	duration time.Duration
}

func InsertHistorianRecords(m *model, fileName string) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		errStr := ""
		records := m.datFileRecords[fileName].FloatRecords
		pointCache := m.datFileRecords[fileName].PointCache
		err := LibFTH.ConvertDatFloatRecordsToPutSnapshots(*records, pointCache)
		if err != nil {
			errStr = err.Error()
		}
		duration := time.Since(start)
		return HistorianInsertMsg{fileName: fileName, err: errStr, duration: duration}
	}
}
