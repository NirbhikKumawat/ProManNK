package ui

import (
	"ProManNK/process"
	"fmt"
	"os"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SortMethod string

const (
	SortPID     SortMethod = "PID"
	SortCommand SortMethod = "COMMAND"
	SortUser    SortMethod = "USER"
	SortCPU     SortMethod = "CPU"
	SortMem     SortMethod = "MEM"
)

var sortMethods = []SortMethod{SortPID, SortCommand, SortUser, SortCPU, SortMem}

var (
	subtleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	pidStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	cmdStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	userStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	statStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("43"))
	selectedText   = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Background(lipgloss.Color("236"))
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Underline(true)
	labelStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	statusBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("62")).Padding(0, 1)
	headerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("229")).Background(lipgloss.Color("57")).Padding(0, 1).Bold(true)
)

type Model struct {
	Processes   []*process.Process
	Visible     []*process.Process
	Expanded    map[int]bool
	cursor      int
	offset      int
	width       int
	height      int
	err         error
	status      string
	display     string
	showDetails bool
	searchBar   textinput.Model
	isSearching bool
	filterQuery string
	sortMethod  SortMethod
	sortAsc     bool
}

type ErrorMsg error
type ClearErrorMsg struct{}
type TickMsg struct{}

func nextSortMethod(current SortMethod) SortMethod {
	for i, m := range sortMethods {
		if m == current {
			return sortMethods[(i+1)%len(sortMethods)]
		}
	}
	return SortPID
}

func prevSortMethod(current SortMethod) SortMethod {
	for i, m := range sortMethods {
		if m == current {
			idx := i - 1
			if idx < 0 {
				idx = len(sortMethods) - 1
			}
			return sortMethods[idx]
		}
	}
	return SortPID
}

func sortTree(nodes []*process.Process, method SortMethod, asc bool) {
	sort.SliceStable(nodes, func(i, j int) bool {
		var less bool
		switch method {
		case SortPID:
			less = nodes[i].PID < nodes[j].PID
		case SortCommand:
			less = strings.ToLower(nodes[i].Command) < strings.ToLower(nodes[j].Command)
		case SortUser:
			less = strings.ToLower(nodes[i].User) < strings.ToLower(nodes[j].User)
		case SortCPU:
			less = nodes[i].CPU < nodes[j].CPU
		case SortMem:
			less = nodes[i].Memory < nodes[j].Memory
		default:
			less = nodes[i].PID < nodes[j].PID
		}
		if !asc {
			return !less
		}
		return less
	})
	for _, n := range nodes {
		if len(n.Children) > 0 {
			sortTree(n.Children, method, asc)
		}
	}
}

func NewModel() *Model {
	processes, err := process.BuildTree()
	if err != nil {
		panic(err)
	}

	ti := textinput.New()
	ti.Placeholder = "Press '/' to search by name or PID..."
	ti.CharLimit = 50
	ti.Width = 30
	ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)
	return &Model{
		cursor:      0,
		offset:      0,
		Processes:   processes,
		Visible:     process.FlattenVisible(processes),
		display:     "",
		Expanded:    make(map[int]bool),
		status:      "ProManNK just started",
		err:         nil,
		searchBar:   ti,
		isSearching: false,
		sortMethod:  SortPID,
		sortAsc:     true,
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
		return TickMsg{}
	})
}

func (m *Model) ApplyFilter() {
	allVisible := process.FlattenVisible(m.Processes)
	if m.filterQuery != "" {
		var filtered []*process.Process
		query := strings.ToLower(m.filterQuery)

		for _, p := range allVisible {
			if strings.Contains(strings.ToLower(p.Command), query) || fmt.Sprintf("%d", p.PID) == query {
				filtered = append(filtered, p)
			}
		}
		m.Visible = filtered
	} else {
		m.Visible = allVisible
	}
}

func (m *Model) UpdateTree() {
	processes, err := process.BuildTree()
	if err != nil {
		panic(err)
	}
	restoreExpansionState(processes, m.Expanded)
	sortTree(processes, m.sortMethod, m.sortAsc)
	m.Processes = processes
	m.ApplyFilter()
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
		usableHeight := m.height - 9
		if m.isSearching {
			switch msg.String() {
			case "esc":
				m.isSearching = false
				m.searchBar.Blur()
				m.filterQuery = ""
				m.cursor = 0
				m.offset = 0
				m.ApplyFilter()
				return m, nil
			case "enter":
				m.isSearching = false
				m.searchBar.Blur()
				return m, nil
			default:
				var cmd tea.Cmd
				m.searchBar, cmd = m.searchBar.Update(msg)
				m.filterQuery = m.searchBar.Value()
				m.offset = 0
				m.cursor = 0
				m.ApplyFilter()
				return m, cmd
			}
		}
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case ">", ".":
			m.sortMethod = nextSortMethod(m.sortMethod)
			m.UpdateTree()
			m.status = fmt.Sprintf("Sorted by %s", m.sortMethod)
		case "<", ",":
			m.sortMethod = prevSortMethod(m.sortMethod)
			m.UpdateTree()
			m.status = fmt.Sprintf("Sorted by %s", m.sortMethod)
		case "r", "R":
			m.sortAsc = !m.sortAsc
			m.UpdateTree()
			m.status = fmt.Sprintf("Sorted by %s", m.sortMethod)
		case "/":
			if !m.showDetails {
				m.isSearching = true
				m.searchBar.Focus()
				m.status = "Typing search query... Press Enter to navigate results, Esc to cancel."
			}
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
		case "d":
			m.showDetails = !m.showDetails
		case "esc":
			m.showDetails = false
		case "E":
			var expandAll func(nodes []*process.Process)
			expandAll = func(nodes []*process.Process) {
				for _, node := range nodes {
					if len(node.Children) > 0 {
						m.Expanded[node.PID] = true
						expandAll(node.Children)
					}
				}
			}
			expandAll(m.Processes)
			restoreExpansionState(m.Processes, m.Expanded)
			m.ApplyFilter()
			if len(m.Visible) > 0 {
				if m.cursor >= len(m.Visible) {
					m.cursor = len(m.Visible) - 1
				}
				if m.offset > m.cursor {
					m.offset = m.cursor
				}
			}
			m.status = "Expanded all process branches."
		case "C":
			m.Expanded = make(map[int]bool)
			restoreExpansionState(m.Processes, m.Expanded)
			m.ApplyFilter()
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
			m.status = "Collapsed all process branches."
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
					restoreExpansionState(m.Processes, m.Expanded)
					m.ApplyFilter()
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
	if m.showDetails {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, m.renderDetails())
	}
	if m.width == 0 {
		return "Initializing..."
	}
	usableHeight := m.height - 9
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

	if m.isSearching || m.filterQuery != "" {
		b.WriteString(" " + m.searchBar.View() + "\n\n")
	} else {
		b.WriteString("\n\n")
	}

	pidCol := "PID"
	cmdCol := "COMMAND"
	userCol := "USER"
	cpuCol := "CPU%"
	memCol := "MEM%"

	arr := "^"
	if !m.sortAsc {
		arr = "v"
	}

	switch m.sortMethod {
	case SortPID:
		pidCol += arr
	case SortCommand:
		cmdCol += arr
	case SortUser:
		userCol += arr
	case SortCPU:
		cpuCol += arr
	case SortMem:
		memCol += arr
	}

	colHeader := fmt.Sprintf("  %-10s %-40s %-15s %-8s %-10s %-8s %-10s %-10s %-12s %-12s %-12s", pidCol, cmdCol, userCol, "STATE", "PPID", "NICE", cpuCol, memCol, "RSS(MB)", "VMS(MB)", "SHR(MB)")
	b.WriteString(subtleStyle.Render(colHeader) + "\n")
	b.WriteString(subtleStyle.Render(strings.Repeat("─", 160)) + "\n")

	if len(m.Visible) == 0 {
		b.WriteString("  No running processes\n")
		linesDrawn := 1
		paddingLines := usableHeight - linesDrawn
		if paddingLines > 0 {
			b.WriteString(strings.Repeat("\n", paddingLines))
		}
		bar := statusBarStyle.Width(m.width).Render("Status: " + m.status)
		helpMenu := subtleStyle.Render(" [↑/k/↓/j] Nav  [</>] Sort  [r] Rev Sort  [Ent] Open/Close Tree  [/] Search  [E/C] Exp/Col  [d] Details  [q] Quit  [s] Pause  [c] Resume  [i] Int  [t] Term  [f] Kill")
		b.WriteString("\n" + bar + "\n" + helpMenu)
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
		if len(fullCmd) > 40 {
			fullCmd = fullCmd[:37] + "..."
		}
		userName := proc.User
		if len(userName) > 15 {
			userName = userName[:14] + "+"
		}
		if m.cursor == i {
			plainRow := fmt.Sprintf("▶ %-10d %-40s %-15s %-8c %-10d %-8d %-10.3f %-10.3f %-12d %-12d %-12d",
				proc.PID,
				fullCmd,
				userName,
				proc.State,
				proc.PPID,
				proc.Nice,
				proc.CPU,
				proc.Memory,
				proc.ResidentM,
				proc.VirtualM,
				proc.SharedM)

			row := selectedText.Width(m.width).Render(plainRow)
			b.WriteString(row + "\n")
		} else {
			pidStr := pidStyle.Render(fmt.Sprintf("%-10d", proc.PID))
			cmdStr := cmdStyle.Render(fmt.Sprintf("%-40s", fullCmd))
			userStr := userStyle.Render(fmt.Sprintf("%-15s", userName))
			stateStr := userStyle.Render(fmt.Sprintf("%-8c", proc.State))
			ppidStr := pidStyle.Render(fmt.Sprintf("%-10d", proc.PPID))
			niceStr := statStyle.Render(fmt.Sprintf("%-8d", proc.Nice))
			cpuStr := statStyle.Render(fmt.Sprintf("%-10.3f", proc.CPU))
			memStr := statStyle.Render(fmt.Sprintf("%-10.3f", proc.Memory))
			rssStr := statStyle.Render(fmt.Sprintf("%-12d", proc.ResidentM))
			vmsStr := statStyle.Render(fmt.Sprintf("%-12d", proc.VirtualM))
			shrStr := statStyle.Render(fmt.Sprintf("%-12d", proc.SharedM))

			row := fmt.Sprintf("  %s %s %s %s %s %s %s %s %s %s %s", pidStr, cmdStr, userStr, stateStr, ppidStr, niceStr, cpuStr, memStr, rssStr, vmsStr, shrStr)
			b.WriteString(row + "\n")
		}
	}

	bar := statusBarStyle.Width(m.width).Render("Status: " + m.status)
	helpMenu := subtleStyle.Render(" [↑/k/↓/j] Nav  [</>] Sort  [r] Rev Sort  [Ent] Open/Close Tree  [/] Search  [E/C] Exp/Col  [d] Details  [q] Quit  [s] Pause  [c] Resume  [i] Int  [t] Term  [f] Kill")
	linesDrawn := end - m.offset
	paddingLines := usableHeight - linesDrawn
	if paddingLines > 0 {
		b.WriteString(strings.Repeat("\n", paddingLines))
	}
	b.WriteString("\n" + bar + "\n" + helpMenu)
	return b.String()
}

func (m *Model) renderDetails() string {
	selected := m.Visible[m.cursor]

	detailBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("212")).
		Padding(1, 2).
		Width(60)

	content := fmt.Sprintf(
		"%s\n\n"+
			"%s %d\n"+
			"%s %d\n"+
			"%s %c\n"+
			"%s %d\n"+
			"%s %s\n\n"+
			"%s\n"+
			"%s %d MB\n"+
			"%s %d MB\n"+
			"%s %d MB\n"+
			"%s %s",
		titleStyle.Render("PROCESS INSPECTOR"),
		labelStyle.Render("PID:      "), selected.PID,
		labelStyle.Render("Parent:   "), selected.PPID,
		labelStyle.Render("State:    "), selected.State,
		labelStyle.Render("Nice:     "), selected.Nice,
		labelStyle.Render("User:     "), selected.User,
		titleStyle.Render("MEMORY USAGE"),
		labelStyle.Render("Virtual:  "), selected.VirtualM,
		labelStyle.Render("Resident: "), selected.ResidentM,
		labelStyle.Render("Shared:   "), selected.SharedM,
		labelStyle.Render("Started:  "), selected.Time.Format("15:04:05"),
	)

	return detailBox.Render(content)
}
