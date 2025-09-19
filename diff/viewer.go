package diff

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/networkteam/shry/ui"
)

type viewerModel struct {
	viewport viewport.Model
	ready    bool
	content  string
}

func newViewerModel(content string) viewerModel {
	return viewerModel{
		viewport: viewport.New(0, 0),
		content:  content,
	}
}

func (m viewerModel) Init() tea.Cmd {
	return nil
}

func (m viewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 3
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m viewerModel) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	headerStyle := ui.TitleStyle.
		Foreground(ui.SecondaryColor).
		Background(ui.PrimaryColor).
		Padding(0, 1)

	header := headerStyle.Render("Diff Viewer")
	footer := ui.HelpStyle.Render("↑/↓: scroll • q/esc: quit • space/b: page up/down")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		m.viewport.View(),
		"",
		footer,
	)
}

func ShowDiff(content string) error {
	m := newViewerModel(content)

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}
