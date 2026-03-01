package ui

import (
	"ProManNK/process"
	"fmt"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	Processes []*process.Process
	Visible   []*process.Process
	Expanded  map[int]bool
	cursor    int
	offset    int
	width     int
	height    int
	err       error
	display   string
}
type ErrorMsg error
type ClearErrorMsg struct{}
type TickMsg struct{}

func NewModel() *Model {
	processes, err := process.BuildTree()
	if err != nil {
		panic(err)
	}
	return &Model{
		cursor:    0,
		offset:    0,
		Processes: processes,
		Visible:   process.FlattenVisible(processes),
		display:   "",
		Expanded:  make(map[int]bool),
	}
}
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}
func (m *Model) UpdateTree() {
	processes, err := process.BuildTree()
	if err != nil {
		panic(err)
	}
	restoreExpansionState(processes, m.Expanded)
	m.Processes = processes
	m.Visible = process.FlattenVisible(processes)
}
func restoreExpansionState(nodes []*process.Process, expandedMap map[int]bool) {
	for _, node := range nodes {
		node.Expanded = expandedMap[node.PID]
		if len(node.Children) > 0 {
			restoreExpansionState(node.Children, expandedMap)
		}
	}
}
func (m *Model) Init() tea.Cmd {
	return tickCmd()
}
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case ErrorMsg:
		m.err = msg
	case ClearErrorMsg:
		m.err = nil
	case TickMsg:
		m.UpdateTree()
		if len(m.Visible) > 0 {
			if m.cursor >= len(m.Visible) {
				m.cursor = len(m.Visible) - 1
			}
			if m.offset > m.cursor {
				m.offset = m.cursor
			}
		} else {
			m.cursor = 0
			m.offset = 0
		}
		return m, tickCmd()
	case tea.KeyMsg:
		usableHeight := m.height - 5
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		case "down", "j":
			if m.cursor < len(m.Visible)-1 {
				m.cursor++
			}
			if m.cursor >= m.offset+usableHeight {
				m.offset = m.cursor - usableHeight + 1
			}
		case "t":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				err := syscall.Kill(selected.PID, syscall.SIGTERM)
				if err != nil {
					panic(err)
				}
			}
		case "f":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				err := syscall.Kill(selected.PID, syscall.SIGKILL)
				if err != nil {
					panic(err)
				}
			}
		case "enter":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if len(selected.Children) > 0 {
					m.Expanded[selected.PID] = !m.Expanded[selected.PID]
					m.UpdateTree()
					if len(m.Visible) > 0 {
						if m.cursor >= len(m.Visible) {
							m.cursor = len(m.Visible) - 1
						}
						if m.offset > m.cursor {
							m.offset = m.cursor
						}
					} else {
						m.cursor = 0
						m.offset = 0
					}
				}
			}
		}
	}
	return m, nil
}
func (m *Model) View() string {
	usableHeight := m.height - 5
	end := m.offset + usableHeight
	if end > len(m.Visible) {
		end = len(m.Visible)
	}
	var b strings.Builder
	b.WriteString("System Processes (Press Enter/Space to expand/collapse):\n\n")
	if len(m.Visible) == 0 {
		b.WriteString("  No running processes\n")
		return b.String()
	}

	for i := m.offset; i < end; i++ {
		proc := m.Visible[i]
		cursorStr := "  "
		if m.cursor == i {
			cursorStr = "> "
		}
		indent := strings.Repeat("  ", proc.Depth)
		folderIcon := "   "
		if len(proc.Children) > 0 {
			if proc.Expanded {
				folderIcon = "[-]"
			} else {
				folderIcon = "[+]"
			}
		}

		line := fmt.Sprintf("%s%s%s %d %s %d %d %s %f %f\n", cursorStr, indent, folderIcon, proc.PID, proc.Command, proc.PPID, proc.Nice, proc.User, proc.CPU, proc.Memory)
		b.WriteString(line)
	}

	return b.String()
}
