package tui

import "github.com/charmbracelet/lipgloss"

// Theme defines a complete color palette for the TUI.
type Theme struct {
	Name string

	// Backgrounds
	Base    lipgloss.Color
	Surface lipgloss.Color
	Overlay lipgloss.Color
	Muted   lipgloss.Color

	// Text
	Text    lipgloss.Color
	Subtle  lipgloss.Color
	Comment lipgloss.Color

	// Accents
	Accent lipgloss.Color
	Green  lipgloss.Color
	Yellow lipgloss.Color
	Red    lipgloss.Color
	Cyan   lipgloss.Color
	Purple lipgloss.Color
}

var (
	TokyoNight = Theme{
		Name:    "tokyo-night",
		Base:    "#1a1b26",
		Surface: "#1f2335",
		Overlay: "#545c7e",
		Muted:   "#565f89",
		Text:    "#c0caf5",
		Subtle:  "#a9b1d6",
		Comment: "#737aa2",
		Accent:  "#7aa2f7",
		Green:   "#9ece6a",
		Yellow:  "#e0af68",
		Red:     "#f7768e",
		Cyan:    "#7dcfff",
		Purple:  "#bb9af7",
	}

	CatppuccinMocha = Theme{
		Name:    "catppuccin",
		Base:    "#1e1e2e",
		Surface: "#181825",
		Overlay: "#313244",
		Muted:   "#6c7086",
		Text:    "#cdd6f4",
		Subtle:  "#bac2de",
		Comment: "#45475a",
		Accent:  "#89b4fa",
		Green:   "#a6e3a1",
		Yellow:  "#f9e2af",
		Red:     "#f38ba8",
		Cyan:    "#89dceb",
		Purple:  "#b4befe",
	}

	Dracula = Theme{
		Name:    "dracula",
		Base:    "#282a36",
		Surface: "#21222c",
		Overlay: "#44475a",
		Muted:   "#6272a4",
		Text:    "#f8f8f2",
		Subtle:  "#bfbfbf",
		Comment: "#44475a",
		Accent:  "#bd93f9",
		Green:   "#50fa7b",
		Yellow:  "#f1fa8c",
		Red:     "#ff5555",
		Cyan:    "#8be9fd",
		Purple:  "#ff79c6",
	}

	GruvboxDark = Theme{
		Name:    "gruvbox",
		Base:    "#282828",
		Surface: "#1d2021",
		Overlay: "#3c3836",
		Muted:   "#7c6f64",
		Text:    "#ebdbb2",
		Subtle:  "#d5c4a1",
		Comment: "#504945",
		Accent:  "#83a598",
		Green:   "#b8bb26",
		Yellow:  "#fabd2f",
		Red:     "#fb4934",
		Cyan:    "#8ec07c",
		Purple:  "#d3869b",
	}
)

var DefaultTheme = TokyoNight

// AllThemes is the ordered cycle used by the in-TUI theme switcher.
var AllThemes = []Theme{TokyoNight, CatppuccinMocha, Dracula, GruvboxDark}

// ThemeByName returns the theme matching the given name, falling back to DefaultTheme.
func ThemeByName(name string) Theme {
	switch name {
	case "catppuccin":
		return CatppuccinMocha
	case "dracula":
		return Dracula
	case "gruvbox":
		return GruvboxDark
	default:
		return TokyoNight
	}
}

// ThemeIndex returns the index of t in AllThemes, or 0 if not found.
func ThemeIndex(t Theme) int {
	for i, theme := range AllThemes {
		if theme.Name == t.Name {
			return i
		}
	}
	return 0
}
