package settings

import (
	"slices"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	terminu "github.com/vulcanshen/terminu-go"

	"github.com/vulcanshen/locku/internal/config"
	"github.com/vulcanshen/locku/internal/saver"
	"github.com/vulcanshen/locku/internal/setup"
)

// Panel [2] (ui.md §1.1): the details of whatever [1]'s cursor is on, one
// value a row, written the moment it is set. A profile is its saver's
// settings and its two colours. A saver is what it is, which profiles are
// of it, and then its DEFAULTS — the same values a profile has, which a
// profile made of it from now on starts as, and which [p] previews;
// changing them changes no profile already made (user, 2026-09-24). A tool
// — tmux, screen — is activate and its file, then its own settings under
// its own names (user, 2026-09-25). Preference is the PIN, the active
// profile and the lock's settings. Every key config.yaml has is a row here,
// except `profiles` and `savers`, which ARE panel [1].
//
// A choice of five or fewer is a radio drawn in place, more a select; a
// switch is a checkbox (components/dialog/form, input/radio, input/checkbox).
// What is only shown — a saver's description, a profile's saver — is a
// disabled row: there, not to be changed, the cursor passes it by.

// about is what [2] says of a saver.
var about = map[string]string{
	saver.KindClock:     "the time and the date, on the LED board",
	saver.KindRunner:    "the offline dino run, jumping by itself, for ever",
	saver.KindBounce:    "the time in a box, bouncing, changing colour",
	saver.KindSnake:     "the Nokia snake, playing itself till the board is full",
	saver.KindTetromino: "falling blocks, playing themselves, full rows going",
	saver.KindPets:      "cats outdoors, up the trees, jumping, napping",
	saver.KindCustom:    "your own program, on a terminal of its own, as the saver",
}

// usualConf is where a tool's file usually is, said in the box's prompt.
var usualConf = map[string]string{"tmux": "~/.tmux.conf", "screen": "~/.screenrc"}

// toolIdle is each tool's own name for its idle time: the row's label,
// and the file's key (user, 2026-09-25: tmux's row says lock-after-time,
// as tmux does). toolBind is its name for binding a key, toolPrefix what
// the key comes after.
var (
	toolIdle   = map[string]string{"tmux": "lock-after-time", "screen": "idle"}
	toolBind   = map[string]string{"tmux": "bind-key", "screen": "bind"}
	toolPrefix = map[string]string{"tmux": "prefix", "screen": "C-a"}
)

// fieldDesc is what Enter does on a row, for its Space menu row.
var fieldDesc = map[string]string{
	"name":                       "this profile",
	"participants":               "one or two, big or small",
	"character":                  "who runs",
	"scene":                      "where it runs",
	"background":                 "day, night, or a day every three minutes",
	"command":                    "the program that draws, as sh -c runs it",
	"layout":                     "a row, or a column of parts",
	"size":                       "small, medium or large digits",
	"font":                       "3x7, or the shorter 3x5",
	"time":                       "the time's shape",
	"date":                       "off, or one of four shapes",
	"speed":                      "how fast it goes",
	"animals":                    "what they are",
	"count":                      "how many cats",
	"bg":                         "the board's ground",
	"fg":                         "the board's lit pixels",
	"PIN":                        "what the lock asks for",
	"profile":                    "the profile the lock shows",
	"show_status":                "user@host and the time, on the lock",
	"pin_prompt_timeout":         "seconds until the PIN box closes; 0 never",
	"wrong_pin_attempts":         "wrong PINs before a cooldown; 0 off",
	"wrong_pin_attempt_cooldown": "the cooldown, in seconds",
	"activate":                   "locku's line in the file, or not",
	"config file path":           "the file locku's line goes into",
	"lock":                       "the whole server, or one session",
	"lock-after-time":            "idle seconds before tmux locks; 0 never",
	"idle":                       "idle seconds before screen locks; 0 never",
	"bind-key":                   "the key after prefix that locks",
	"bind":                       "the key after C-a that locks",
}

// readOnly is a row only shown.
func readOnly(label, value string) *terminu.Field {
	return &terminu.Field{Label: label, Value: func() string { return value }, Disabled: func() bool { return true }}
}

// detailLabel is [2]'s name: the item's own (user, 2026-09-25: no kind, no
// state).
func (s *Settings) detailLabel() string {
	switch s.cur.kind {
	case kindProfile:
		return s.cfg.Profiles[s.cur.ref].Name
	case kindSaver:
		return saver.Kinds[s.cur.ref]
	case kindTool:
		return tools[s.cur.ref]
	}
	return "preference"
}

// fields is [2] for the item under [1]'s cursor.
func (s *Settings) fields() []*terminu.Field {
	switch s.cur.kind {
	case kindSaver:
		kind := saver.Kinds[s.cur.ref]
		var names []string
		for _, p := range s.cfg.Profiles {
			if p.Saver == kind {
				names = append(names, p.Name)
			}
		}
		used := strings.Join(names, ", ")
		if used == "" {
			used = "none yet"
		}
		out := []*terminu.Field{
			readOnly("saver", kind),
			readOnly("what", about[kind]),
			readOnly("profiles", used),
			readOnly("defaults", "for profiles made of it from now on"),
		}
		p, _ := s.subject()
		return append(out, s.profileFields(p)...)
	case kindProfile:
		p, _ := s.subject()
		name := &terminu.Field{Label: "name", Value: func() string { return p.Name }, Edit: func() tea.Cmd { return s.rename(s.cur.ref) }}
		// The saver is the profile's class: shown, not changed — a profile
		// of another saver is a new profile (user, 2026-09-24).
		out := []*terminu.Field{name, readOnly("saver", p.Saver)}
		return append(out, s.profileFields(p)...)
	case kindTool:
		return s.toolFields()
	}
	return s.preferenceFields()
}

// profileFields is a saver's own settings for p, then its colours: the
// clock's shapes and size, the run's runner and scene, the custom saver's
// command, the speeds (user, 2026-10-06 and 2026-10-07), the pets' animals,
// count and scene (2026-10-07, 2026-10-08).
func (s *Settings) profileFields(p config.Profile) []*terminu.Field {
	set := func(f func(*config.Profile, string)) func(string) tea.Cmd {
		return func(v string) tea.Cmd { return s.edit(func(p *config.Profile) { f(p, v) }) }
	}
	var out []*terminu.Field
	switch p.Saver {
	case saver.KindBounce:
		out = []*terminu.Field{
			s.choice("speed", saver.Speeds, p.Speed, set(func(p *config.Profile, v string) { p.Speed = v })),
			s.choice("time", saver.BounceTimes, p.Time, set(func(p *config.Profile, v string) { p.Time = v })),
		}
	case saver.KindSnake, saver.KindTetromino:
		out = []*terminu.Field{s.choice("speed", saver.Speeds, p.Speed, set(func(p *config.Profile, v string) { p.Speed = v }))}
	case saver.KindPets:
		counts := make([]string, len(saver.PetCounts))
		for i, n := range saver.PetCounts {
			counts[i] = itoa(n)
		}
		out = []*terminu.Field{
			s.choice("animals", saver.PetAnimals, p.Animals, set(func(p *config.Profile, v string) { p.Animals = v })),
			s.choice("count", counts, itoa(p.Count), set(func(p *config.Profile, v string) { p.Count, _ = strconv.Atoi(v) })),
			s.choice("scene", saver.PetScenes, p.Scene, set(func(p *config.Profile, v string) { p.Scene = v })),
		}
	case saver.KindCustom:
		// The program, as sh -c runs it; none yet is said in grey.
		out = []*terminu.Field{{Label: "command", Value: func() string { return p.Command }, Empty: "not set",
			Edit: func() tea.Cmd {
				return s.app.Text(terminu.TextInput{Title: "Edit", Subject: "command", Glyph: glyphConsole, Name: "command",
					Prompt: "The program that draws, as sh -c runs it; empty for none", Verb: "save", Old: p.Command,
					Done: set(func(p *config.Profile, v string) { p.Command = v })})
			}}}
	case saver.KindRunner:
		// No size: the run is drawn as large as the terminal allows
		// (user, 2026-09-24).
		out = []*terminu.Field{
			s.choice("participants", saver.Participants, p.Participants, set(func(p *config.Profile, v string) { p.Participants = v })),
			s.choice("character", saver.Characters, p.Character, set(func(p *config.Profile, v string) { p.Character = v })),
			s.choice("scene", saver.Scenes, p.Scene, set(func(p *config.Profile, v string) { p.Scene = v })),
			s.choice("background", saver.Backgrounds, p.Background, set(func(p *config.Profile, v string) { p.Background = v })),
		}
	default:
		out = []*terminu.Field{
			s.choice("layout", saver.Layouts, p.Layout, set(func(p *config.Profile, v string) { p.Layout = v })),
			s.choice("size", saver.Sizes, p.Size, set(func(p *config.Profile, v string) { p.Size = v })),
			s.choice("font", saver.Fonts, p.Font, set(func(p *config.Profile, v string) { p.Font = v })),
			s.choice("time", saver.TimeFormats, p.Time, set(func(p *config.Profile, v string) { p.Time = v })),
			s.choice("date", saver.DateFormats, p.Date, set(func(p *config.Profile, v string) { p.Date = v })),
		}
	}
	// No colours: custom (user, 2026-09-25), and a saver with its own
	// (2026-10-06).
	if saver.Coloured(p.Saver) {
		c := p.Colours()
		out = append(out,
			s.colour("bg", c.BG, set(func(p *config.Profile, v string) { p.BG = v })),
			s.colour("fg", c.FG, set(func(p *config.Profile, v string) { p.FG = v })))
	}
	return out
}

// choice is a value out of options: a radio drawn in place for five or
// fewer, a select past that (components/dialog/form).
func (s *Settings) choice(label string, options []string, current string, set func(string) tea.Cmd) *terminu.Field {
	at := slices.Index(options, current)
	if len(options) <= 5 {
		f := &terminu.Field{Label: label, Options: options, Selected: at}
		f.OnChange = func() tea.Cmd { return set(options[f.Selected]) }
		return f
	}
	return &terminu.Field{Label: label, Value: func() string { return current },
		Edit: func() tea.Cmd {
			return s.app.Select(terminu.Select{Title: label, Subject: s.detailLabel(), Glyph: glyphTag, Options: options, Selected: at,
				Done: func(i int) tea.Cmd { return set(options[i]) }})
		}}
}

// swatch is a colour as a value shows it: a block of it, then its hex
// (components/input/color-picker, the value).
func swatch(hex string) string {
	return terminu.Fg(terminu.Hex(hex)).Render("███") + terminu.Fg(terminu.Text).Render(" "+strings.ToUpper(hex))
}

// colour is one of a profile's two colours, chosen in the color picker:
// old and new side by side, Esc leaves the old (components/input/color-picker).
func (s *Settings) colour(label, hex string, set func(string) tea.Cmd) *terminu.Field {
	return &terminu.Field{Label: label, Value: func() string { return swatch(hex) },
		Edit: func() tea.Cmd {
			return s.app.PickColor(terminu.ColorPicker{Title: label, Subject: s.detailLabel(), Glyph: glyphPalette, Value: terminu.Hex(hex),
				Done: func(c terminu.Color) tea.Cmd { return set(c.String()) }})
		}}
}

// switchField is an on / off: a checkbox drawn in place (components/input/checkbox).
func switchField(label string, on bool) *terminu.Field {
	return &terminu.Field{Label: label, Options: []string{"on"}, Multi: true, Checked: []bool{on}}
}

// number is a whole number of 0 or more, edited in a number box; zeroOff
// shows 0 as "0 (off)".
func (s *Settings) number(label, prompt string, value int, zeroOff bool, set func(int) tea.Cmd) *terminu.Field {
	return &terminu.Field{Label: label,
		Value: func() string {
			if zeroOff && value == 0 {
				return "0 (off)"
			}
			return itoa(value)
		},
		Edit: func() tea.Cmd {
			return s.app.Number(terminu.NumberInput{Title: "Edit", Subject: label, Glyph: glyphTimer, Name: label,
				Prompt: prompt, Verb: "save", Old: itoa(value),
				Done: func(v string) tea.Cmd {
					n, err := strconv.Atoi(v)
					if err != nil {
						return nil
					}
					return set(n)
				}})
		}}
}

func (s *Settings) preferenceFields() []*terminu.Field {
	pin := &terminu.Field{Label: "PIN", Empty: "not set", Edit: s.editPIN,
		Value: func() string {
			if s.cfg.HasPIN() {
				return "••••••••" // set: eight, whatever its length (components/input/password)
			}
			return ""
		}}
	names := make([]string, len(s.cfg.Profiles))
	for i, p := range s.cfg.Profiles {
		names[i] = p.Name
	}
	profile := s.choice("profile", names, s.cfg.Profile, func(v string) tea.Cmd {
		return s.change(func() { s.cfg.Profile = v })
	})
	status := switchField("show_status", s.cfg.ShowStatus)
	status.OnChange = func() tea.Cmd {
		on := status.Checked[0]
		return s.change(func() { s.cfg.ShowStatus = on })
	}
	def := config.Default()
	return []*terminu.Field{
		pin,
		profile,
		status,
		s.number("pin_prompt_timeout", "Seconds without a key before the PIN box closes, 0 or more; 0 never, "+itoa(def.PINPromptTimeout)+" by default",
			s.cfg.PINPromptTimeout, false, func(n int) tea.Cmd { return s.change(func() { s.cfg.PINPromptTimeout = n }) }),
		s.number("wrong_pin_attempts", "Wrong PINs in a row before a cooldown, 0 or more; 0 off, "+itoa(def.WrongPINAttempts)+" by default",
			s.cfg.WrongPINAttempts, true, func(n int) tea.Cmd { return s.change(func() { s.cfg.WrongPINAttempts = n }) }),
		s.number("wrong_pin_attempt_cooldown", "Seconds the cooldown lasts, 1 or more; "+itoa(def.WrongPINCooldown)+" by default",
			s.cfg.WrongPINCooldown, false, func(n int) tea.Cmd {
				if n == 0 {
					n = def.WrongPINCooldown
				}
				return s.change(func() { s.cfg.WrongPINCooldown = n })
			}),
	}
}

// tool is the tool under the cursor and its settings.
func (s *Settings) tool() (name string, t config.Tool) {
	name = tools[s.cur.ref]
	return name, s.cfg.Tool(name)
}

// bindOf is the key the tool binds to the lock — tmux's bind-key,
// screen's bind — which a Tool does not carry.
func (s *Settings) bindOf(name string) string {
	if name == tools[toolScreen] {
		return s.cfg.Screen.Bind
	}
	return s.cfg.Tmux.BindKey
}

// toolFields is a tool's [2]: locku's own two rows — activate, read off
// the file each time, and the file — then the tool's own settings under
// the tool's own names (user, 2026-09-25).
func (s *Settings) toolFields() []*terminu.Field {
	name, t := s.tool()
	installed := setup.Installed(t.Conf)
	activate := switchField("activate", installed)
	activate.Disabled = func() bool { return t.Conf == "" }
	activate.OnChange = func() tea.Cmd {
		activate.Checked[0] = installed // it turns once it is done, after the confirm
		return s.askActivate(name, !installed)
	}
	conf := &terminu.Field{Label: "config file path", Value: func() string { return t.Conf }, Empty: "not set", Path: true,
		Edit: func() tea.Cmd {
			return s.app.Text(terminu.TextInput{Title: "Edit", Subject: "config file path", Glyph: glyphFile, Name: "config file path",
				Prompt: "The file " + name + "'s line goes into, usually " + usualConf[name] + "; empty for none", Verb: "save", Old: t.Conf,
				Check: func(v string) error {
					if _, ok := config.AbsPath(v); v != "" && !ok {
						return errNotAbs
					}
					return nil
				},
				Done: func(v string) tea.Cmd {
					cmd := s.change(func() {
						nt := s.cfg.Tool(name)
						nt.Conf = v
						s.cfg.SetTool(name, nt)
					})
					return tea.Batch(cmd, s.moveConf(name, t.Conf, installed))
				}})
		}}
	out := []*terminu.Field{activate, conf}
	// tmux alone chooses its lock: the server's, or this session's (user,
	// 2026-09-25).
	if name == tools[toolTmux] {
		out = append(out, s.choice("lock", config.TmuxLocks, s.cfg.Tmux.Lock, func(v string) tea.Cmd {
			cmd := s.change(func() { s.cfg.Tmux.Lock = v })
			return tea.Batch(cmd, s.resync(name, installed))
		}))
	}
	out = append(out, s.number(toolIdle[name], "Idle seconds before "+name+" locks by itself, 0 or more; 0 never, "+itoa(config.DefaultIdle)+" by default",
		t.Idle, true, func(n int) tea.Cmd {
			cmd := s.change(func() {
				t := s.cfg.Tool(name)
				t.Idle = n
				s.cfg.SetTool(name, t)
			})
			return tea.Batch(cmd, s.resync(name, installed))
		}))
	// Each binds a key, under its own name for it, as it spells it; none
	// is none (user, 2026-09-25).
	prompt := "The key after " + toolPrefix[name] + " that locks, as " + name + " spells it — l, C-l; empty binds none"
	example := "e.g. l or C-l"
	if name == tools[toolScreen] {
		prompt = "The key after C-a that locks, as screen spells it — l, ^L; empty binds none, and C-a x locks anyway"
		example = "e.g. l or ^L"
	}
	bind := s.bindOf(name)
	out = append(out, &terminu.Field{Label: toolBind[name], Value: func() string { return bind }, Empty: "none",
		Edit: func() tea.Cmd {
			return s.app.Text(terminu.TextInput{Title: "Edit", Subject: toolBind[name], Glyph: glyphKey, Name: toolBind[name],
				Prompt: prompt, Verb: "save", Old: bind,
				// One key as the tool names it: a space would make it two
				// words on the line, a # the rest of the line a comment.
				Check: func(v string) error {
					if strings.ContainsAny(v, " #") {
						return oneKeyErr(toolBind[name], example)
					}
					return nil
				},
				Done: func(v string) tea.Cmd {
					cmd := s.change(func() {
						if name == tools[toolScreen] {
							s.cfg.Screen.Bind = v
						} else {
							s.cfg.Tmux.BindKey = v
						}
					})
					return tea.Batch(cmd, s.resync(name, installed))
				}})
		}})
	return out
}

// detailActions is what can be done on [2] (ux.md §A.1): Enter on the row
// under the cursor, then the panel's — Preview, and New on a saver.
func (s *Settings) detailActions() []terminu.Action {
	var out []terminu.Action
	if f := s.vals.Current(); f != nil && !(f.Disabled != nil && f.Disabled()) {
		hint := s.vals.EnterHint()
		label := map[string]string{"edit": "Edit", "choose": "Choose", "toggle": "Toggle"}[hint]
		if label != "" {
			desc := fieldDesc[f.Label]
			if f.Label == "scene" {
				if p, ok := s.subject(); ok && p.Saver == saver.KindPets {
					desc = "where they are"
				}
			}
			out = append(out, terminu.Action{Key: "Enter", Label: label, Desc: desc, Scope: terminu.ItemScope, Hint: hint,
				Do: func() tea.Cmd {
					_, cmd := s.vals.Key("Enter", tea.KeyMsg{Type: tea.KeyEnter})
					return cmd
				}})
		}
	}
	switch s.cur.kind {
	case kindSaver:
		kind := saver.Kinds[s.cur.ref]
		out = append(out, s.newAction(kind, terminu.PanelScope),
			terminu.Action{Key: "P", Label: "Preview", Desc: "the lock, showing this saver with its defaults", Scope: terminu.PanelScope,
				Do: func() tea.Cmd { return s.previewSaver(kind) }})
	case kindProfile:
		ref := s.cur.ref
		desc := "the lock, showing this profile"
		if s.cfg.Profiles[ref].Saver == saver.KindCustom {
			desc = "the program, on the terminal, until a key"
		}
		out = append(out, terminu.Action{Key: "P", Label: "Preview", Desc: desc, Scope: terminu.PanelScope,
			Do: func() tea.Cmd { return s.previewProfile(ref) }})
	default:
		// Preference, a tool: the lock as it is, the active profile (user,
		// 2026-09-24).
		out = append(out, terminu.Action{Key: "P", Label: "Preview", Desc: "the lock, showing the active profile", Scope: terminu.PanelScope,
			Do: s.previewActive})
	}
	return out
}

// glossary is what each row of preference's [2], or a tool's, means: the
// last section of its ? (user, 2026-09-25 — and, 2026-09-26 and
// 2026-09-27, in place of the key reference there; the lib puts it after
// the keys).
// TODO(tdp LK3): where a settings panel's "what each row is" belongs.
func (s *Settings) glossary() []terminu.KeySection {
	switch s.cur.kind {
	case kindPreference:
		return []terminu.KeySection{{Title: "what each row is", Rows: []terminu.KeyRow{
			{Key: "PIN", Desc: "what the lock asks for; with none, any key unlocks"},
			{Key: "profile", Desc: "the profile the lock shows"},
			{Key: "show_status", Desc: "user@host and the time, on the lock's last row"},
			{Key: "pin_prompt_timeout", Desc: "seconds without a key before the PIN box closes; 0 never"},
			{Key: "wrong_pin_attempts", Desc: "wrong PINs in a row before a cooldown; 0 off"},
			{Key: "wrong_pin_attempt_cooldown", Desc: "seconds the cooldown lasts"},
		}}}
	case kindTool:
		name := tools[s.cur.ref]
		rows := []terminu.KeyRow{
			{Key: "activate", Desc: "on: the file reads locku's own, " + setup.Own(name) + ", and every row here is written into that the moment it changes; off: it does not. [Enter] turns it, after a confirm"},
			{Key: "config file path", Desc: "the file locku's line, one that reads locku's own, is written into; ~/ allowed"},
		}
		if name == tools[toolTmux] {
			// About how much a lock covers — not what runs it, which is
			// every trigger alike (user, 2026-09-25).
			rows = append(rows, terminu.KeyRow{Key: "lock", Desc: "how much a lock covers. lock-server: the whole server — every client, and whoever attaches to any session while it is locked. lock-session: only the session that locked — its clients, and whoever attaches to it; the other sessions go on as they were"})
		}
		rows = append(rows, terminu.KeyRow{Key: toolIdle[name], Desc: "idle seconds before " + name + " locks by itself; 0 never"})
		if name == tools[toolTmux] {
			rows = append(rows, terminu.KeyRow{Key: "bind-key", Desc: "the key after prefix that runs the lock, as tmux spells it — l, C-l, F12; empty binds none"})
		} else {
			// screen has its own key for the lock, and no server: the lock
			// is LOCKPRG in the shell's environment, so it lives in the
			// shell rc, not the screenrc (measured 2026-09-24).
			rows = append(rows,
				terminu.KeyRow{Key: "bind", Desc: "the key after C-a that runs the lock, as screen's bind spells it — l, ^L; empty binds none, and C-a x — screen's own key for it — locks anyway"},
				terminu.KeyRow{Key: "LOCKPRG", Desc: "the lock itself, locku by its absolute path: not a row, and not in this file — screen reads it from the environment of the shell that started it, so activate puts it into the shell rc, and a new shell has it; a screen already running takes idle and the key at once, but its lock is the shell's it was attached from until it is detached and attached again from a new shell"})
		}
		return []terminu.KeySection{{Title: "what each row is", Rows: rows}}
	}
	return nil
}
