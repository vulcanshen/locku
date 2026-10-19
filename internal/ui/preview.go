package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/custom"
)

// RunPreview is the settings screen's preview (ui.md §2.1): the lock on
// cfg's active profile, on the whole terminal, until any key — no PIN
// asked. The settings screen has handed the terminal over (tea.Exec), its
// widths measured already by the lib.
func RunPreview(cfg config.Config) error { return runPreview(newLock(cfg, "", true)) }

// RunPreviewWord is a preview of a custom saver's ending: the word on the
// board, the note on the status row.
func RunPreviewWord(cfg config.Config, o custom.Outcome) error {
	return runPreview(newLock(cfg, "", true).withWord(o.Word, o.Note))
}

func runPreview(m *LockModel) error {
	_, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(ScreenOutput{os.Stdout})).Run()
	return err
}
