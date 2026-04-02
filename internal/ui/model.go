package ui

import (
	"ProManNK/process"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	pidStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	cmdStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	userStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("43"))

	selectedText = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(lipgloss.Color("236"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Padding(0, 1).
			Bold(true)
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
	status    string
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
		status:    "ProManNK just started",
		err:       nil,
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
		usableHeight := m.height - 7
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
		case "i":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if selected.PID == 1 {
					m.status = "Operation aborted: Refusing to interrupt init system."
					return m, nil
				}
				err := syscall.Kill(selected.PID, syscall.SIGINT)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot interrupt PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to interrupt %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Sent SIGINT (Interrupt) to PID %d", selected.PID)
				}
			}
		case "s":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if selected.PID == 1 {
					m.status = "Operation aborted: Refusing to pause init system."
					return m, nil
				}
				err := syscall.Kill(selected.PID, syscall.SIGSTOP)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot pause PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to pause %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Sent SIGSTOP (Paused) to PID %d", selected.PID)
				}
			}

		case "c":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				// We don't need to block resuming PID 1, but it shouldn't be paused anyway
				err := syscall.Kill(selected.PID, syscall.SIGCONT)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot resume PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to resume %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Sent SIGCONT (Resumed) to PID %d", selected.PID)
				}
			}
		case "t":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if selected.PID == 1 {
					m.status = "Operation aborted: Refusing to kill self or init system."
					return m, nil
				}
				err := syscall.Kill(selected.PID, syscall.SIGTERM)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot kill PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to kill %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Successfully killed PID %d", selected.PID)
				}
			}
		case "f":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if selected.PID == 1 {
					m.status = "Operation aborted: Refusing to kill self or init system."
					return m, nil
				}
				err := syscall.Kill(selected.PID, syscall.SIGKILL)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot kill PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to kill %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Successfully killed PID %d", selected.PID)
				}
			}
		case "T":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if selected.PID == 1 {
					m.status = "Operation aborted: Refusing to kill self or init system."
					return m, nil
				}
				err := process.CascadingGracefulKill(selected)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot kill PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to kill %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Successfully killed PID %d", selected.PID)
				}
			}
		case "F":
			if len(m.Visible) > 0 {
				selected := m.Visible[m.cursor]
				if selected.PID == 1 {
					m.status = "Operation aborted: Refusing to kill self or init system."
					return m, nil
				}
				err := process.CascadingForcefulKill(selected)
				if err != nil {
					if os.IsPermission(err) || err == syscall.EPERM {
						m.status = fmt.Sprintf("Permission Denied: Cannot kill PID %d. Run with sudo.", selected.PID)
					} else {
						m.status = fmt.Sprintf("Failed to kill %d: %v", selected.PID, err)
					}
				} else {
					m.status = fmt.Sprintf("Successfully killed PID %d", selected.PID)
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
	if m.width == 0 {
		return "Initializing..."
	}
	usableHeight := m.height - 7
	if usableHeight < 1 {
		usableHeight = 1
	}
	end := m.offset + usableHeight
	if end > len(m.Visible) {
		end = len(m.Visible)
	}
	var b strings.Builder
	title := " 🌲 ProManNK : Visual Process Orchestrator "
	b.WriteString(headerStyle.Render(title) + "\n\n")
	colHeader := fmt.Sprintf("  %-8s %-35s %-12s %-8s %-8s", "PID", "COMMAND", "USER", "CPU%", "MEM%")
	b.WriteString(subtleStyle.Render(colHeader) + "\n")
	b.WriteString(subtleStyle.Render(strings.Repeat("─", 78)) + "\n")
	if len(m.Visible) == 0 {
		b.WriteString("  No running processes\n")
		return b.String()
	}
	for i := m.offset; i < end; i++ {
		proc := m.Visible[i]
		indent := strings.Repeat("  ", proc.Depth)
		folderIcon := "   "
		if len(proc.Children) > 0 {
			if proc.Expanded {
				folderIcon = "[-]"
			} else {
				folderIcon = "[+]"
			}
		}
		treePrefix := fmt.Sprintf("%s%s", indent, folderIcon)
		fullCmd := treePrefix + proc.Command
		if len(fullCmd) > 35 {
			fullCmd = fullCmd[:32] + "..."
		}
		userName := proc.User
		if len(userName) > 12 {
			userName = userName[:11] + "+"
		}
		if m.cursor == i {
			plainRow := fmt.Sprintf("▶ %-8d %-35s %-12s %-8.1f %-8.1f",
				proc.PID,
				fullCmd,
				userName,
				proc.CPU,
				proc.Memory)

			row := selectedText.Width(m.width).Render(plainRow)
			b.WriteString(row + "\n")
		} else {
			pidStr := pidStyle.Render(fmt.Sprintf("%-8d", proc.PID))
			cmdStr := cmdStyle.Render(fmt.Sprintf("%-35s", fullCmd))
			userStr := userStyle.Render(fmt.Sprintf("%-12s", userName))
			cpuStr := statStyle.Render(fmt.Sprintf("%-8.1f", proc.CPU))
			memStr := statStyle.Render(fmt.Sprintf("%-8.1f", proc.Memory))

			row := fmt.Sprintf("  %s %s %s %s %s", pidStr, cmdStr, userStr, cpuStr, memStr)
			b.WriteString(row + "\n")
		}
	}
	bar := statusBarStyle.Width(m.width).Render("Status: " + m.status)
	helpMenu := subtleStyle.Render(" [↑/k/↓/j] Nav  [Enter] Tree  [s] Pause  [c] Resume  [i] Int  [t] Term  [f] Kill  [q] Quit")
	linesDrawn := end - m.offset
	paddingLines := usableHeight - linesDrawn
	if paddingLines > 0 {
		b.WriteString(strings.Repeat("\n", paddingLines))
	}
	b.WriteString("\n" + bar + "\n" + helpMenu)
	return b.String()
}
