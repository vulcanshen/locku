package ui

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/vulcanshen/locku/internal/version"
)

type splashTickMsg struct{}
type splashIdentityMsg struct{} // fires the name + tagline together
type splashHintMsg struct{}

// splashModel renders the locku logo as a hidden easter egg, a sibling of
// kbu's, filu's, sshu's and webu's splashes. The u-family mark is a navy U
// wrapping a gold figure — here a padlock — and it reveals in that order:
// the background sheet, the lock, then the U frame rising around it.
//
// The family's key is V, and it is V here too, from either panel of the
// settings screen. It is the one place the pixel style is drawn in three
// colours: the lock screen's board is two, and the user's (ui.md §5).
type splashModel struct {
	active          bool
	pixelOrder      []int    // reveal order across all stages
	orderColor      []string // colour for pixelOrder[i] (parallel)
	stageEnds       []int    // cumulative pixel count at each stage's end
	stageStep       []int    // pixels revealed per tick within each stage
	beatsDone       int      // inter-stage holds already taken
	revealedCount   int
	identityVisible bool // "locku" line
	versionVisible  bool // the version line
	taglineVisible  bool // the tagline line
	hintVisible     bool // the Esc hint
}

func newSplashModel() splashModel { return splashModel{} }

func (m splashModel) isActive() bool { return m.active }

// show activates the splash and returns the first animation tick. Reveal
// stages, each held apart by a beat: (1) background — a dark sheet,
// row-major top-to-bottom sweep; (2) the lock, scattered in; (3) the U
// frame (navy), bottom-to-top so it rises from the base around the mark.
// Then a hold reveals the name + version + tagline, and a final hold the
// Esc hint.
func (m *splashModel) show() tea.Cmd {
	m.active = true
	m.revealedCount = 0
	m.beatsDone = 0
	m.identityVisible = false
	m.versionVisible = false
	m.taglineVisible = false
	m.hintVisible = false

	rows, cols := len(logoPixels), len(logoPixels[0])
	// Background pass covers EVERY cell so the sheet fills solid; the later
	// passes paint over it (overwrite, not gaps).
	var bg []int
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			bg = append(bg, r*cols+c)
		}
	}
	// band returns one code's pixels in the order given: top-to-bottom,
	// shuffled, or bottom-to-top.
	band := func(b byte, order string) []int {
		var px []int
		for r := 0; r < rows; r++ {
			for c := 0; c < cols; c++ {
				if logoPixels[r][c] == b {
					px = append(px, r*cols+c)
				}
			}
		}
		switch order {
		case "shuffle":
			rand.Shuffle(len(px), func(i, j int) { px[i], px[j] = px[j], px[i] })
		case "rise":
			for i, j := 0, len(px)-1; i < j; i, j = i+1, j-1 {
				px[i], px[j] = px[j], px[i]
			}
		}
		return px
	}

	m.pixelOrder, m.orderColor, m.stageEnds, m.stageStep = nil, nil, nil, nil
	addStage := func(px []int, color string, step int) {
		m.pixelOrder = append(m.pixelOrder, px...)
		for range px {
			m.orderColor = append(m.orderColor, color)
		}
		m.stageEnds = append(m.stageEnds, len(m.pixelOrder))
		m.stageStep = append(m.stageStep, step)
	}
	addStage(bg, logoBg, cols) // one full row per tick
	addStage(band('L', "shuffle"), logoGold, 3)
	addStage(band('U', "rise"), logoNavy, 3) // the frame rises from the base

	return tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
}

// locku logo — the u-family mark: D = background sheet, U = navy frame,
// L = the gold padlock: a shackle over a body with a keyhole.
var logoPixels = [21]string{
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
	"DDUUUDDDDDDDDDDDDDDDUUUDD",
	"DDDUUDDDDDDDDDDDDDDDUUDDD",
	"DDDUUDDDDDLLLLLDDDDDUUDDD",
	"DDDUUDDDDLDDDDDLDDDDUUDDD",
	"DDDUUDDDDLDDDDDLDDDDUUDDD",
	"DDDUUDDDDLDDDDDLDDDDUUDDD",
	"DDDUUDDDDLDDDDDLDDDDUUDDD",
	"DDDUUDDLLLLLLLLLLLDDUUDDD",
	"DDDUUDDLLLLLLLLLLLDDUUDDD",
	"DDDUUDDLLLLLLLLLLLDDUUDDD",
	"DDDUUDDLLLLDDDLLLLDDUUDDD",
	"DDDUUDDLLLLDDDLLLLDDUUDDD",
	"DDDUUDDLLLLLDLLLLLDDUUDDD",
	"DDDUUDDLLLLLDLLLLLDDUUDDD",
	"DDDUUDDLLLLLLLLLLLDDUUDDD",
	"DDDUUDDLLLLLLLLLLLDDUUDDD",
	"DDDUUDDDDDDDDDDDDDDDUUDDD",
	"DDUUUUUUUUUUUUUUUUUUUUUDD",
	"DUUUUUUUUUUUUUUUUUUUUUUUD",
	"DDDDDDDDDDDDDDDDDDDDDDDDD",
}

const (
	logoBg   = "#313244" // background sheet (catppuccin surface0)
	logoNavy = "#205090" // U frame (the family icon)
	logoGold = "#f2b753" // the gold mark (the family icon)
)

func (m splashModel) render(width, height int) string {
	if !m.active {
		return ""
	}
	cols := len(logoPixels[0])
	cellColor := make([]string, len(logoPixels)*cols)
	for i := 0; i < m.revealedCount; i++ {
		cellColor[m.pixelOrder[i]] = m.orderColor[i]
	}
	var logoLines []string
	for r := 0; r < len(logoPixels); r++ {
		var line strings.Builder
		for c := 0; c < cols; c++ {
			if color := cellColor[r*cols+c]; color != "" {
				line.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(pixelCell))
			} else {
				line.WriteString("  ")
			}
		}
		logoLines = append(logoLines, line.String())
	}
	logo := strings.Join(logoLines, "\n")

	// Caption space is always reserved so the logo does not shift when
	// text appears.
	logoW := cols * 2
	name := lipgloss.NewStyle().Foreground(focusColor).Bold(true)
	line := lipgloss.NewStyle().Foreground(focusColor)
	dim := lipgloss.NewStyle().Foreground(dimColor)
	identityText, versionText, taglineText, hintText := " ", " ", " ", " "
	devLabelText, devMailText := " ", " "
	if m.identityVisible {
		identityText = name.Render("locku")
	}
	if m.versionVisible {
		versionText = line.Render(version.Display())
	}
	if m.taglineVisible {
		taglineText = line.Render("A screensaver with a PIN, for the terminal")
	}
	if m.hintVisible {
		hintText = dim.Render("Press Esc to close")
		devLabelText = dim.Render("developed by")
		devMailText = dim.Render("vulcan.shen.2304@gmail.com")
	}
	caption := "\n\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, identityText) + "\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, versionText) + "\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, taglineText) + "\n\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, devLabelText) + "\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, devMailText) + "\n\n" +
		lipgloss.PlaceHorizontal(logoW, lipgloss.Center, hintText)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, logo+caption)
}

// update handles key events and animation ticks while the splash is active.
func (m splashModel) update(msg tea.Msg) (splashModel, tea.Cmd) {
	if !m.active {
		return m, nil
	}
	switch msg.(type) {
	case tea.KeyMsg:
		// Any key dismisses the easter egg.
		m = splashModel{}
	case splashTickMsg:
		// Hold once at each stage boundary before the next begins.
		if m.beatsDone < len(m.stageEnds)-1 && m.revealedCount == m.stageEnds[m.beatsDone] {
			m.beatsDone++
			return m, tea.Tick(250*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
		}
		if m.revealedCount < len(m.pixelOrder) {
			stage := 0
			for stage < len(m.stageEnds)-1 && m.revealedCount >= m.stageEnds[stage] {
				stage++
			}
			newCount := m.revealedCount + m.stageStep[stage]
			if newCount > m.stageEnds[stage] {
				newCount = m.stageEnds[stage]
			}
			m.revealedCount = newCount
			return m, tea.Tick(10*time.Millisecond, func(time.Time) tea.Msg { return splashTickMsg{} })
		}
		if !m.identityVisible {
			return m, tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg { return splashIdentityMsg{} })
		}
	case splashIdentityMsg:
		m.identityVisible = true
		m.versionVisible = true
		m.taglineVisible = true
		return m, tea.Tick(500*time.Millisecond, func(time.Time) tea.Msg { return splashHintMsg{} })
	case splashHintMsg:
		m.hintVisible = true
	}
	return m, nil
}
