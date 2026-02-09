package theme

// thRegisterBuiltins registers all built-in themes in the registry.
func thRegisterBuiltins() {
	for _, t := range []Theme{
		thDefaultTheme(),
		thGruvboxTheme(),
		thNordTheme(),
		thCatppuccinTheme(),
		thDraculaTheme(),
		thTokyoNightTheme(),
	} {
		thRegister(t)
	}
}

// thDefaultTheme returns the dark neutral theme with purple accent.
func thDefaultTheme() Theme {
	return Theme{
		Name:       "default",
		Background: "#1e1e1e",
		Foreground: "#d4d4d4",
		Dim:        "#6b6b6b",
		Accent:     "#7C3AED",

		Border:      "#3e3e3e",
		BorderFocus: "#7C3AED",
		Title:       "#d4d4d4",

		StatusOK:      "#4ec970",
		StatusWarn:    "#e5c07b",
		StatusError:   "#e06c75",
		StatusUnknown: "#6b6b6b",

		GaugeFilled: "#4ec970",
		GaugeEmpty:  "#3e3e3e",
		GaugeWarn:   "#e5c07b",
		GaugeCrit:   "#e06c75",

		ChartLine: "#7C3AED",
		ChartFill: "#5b21b6",
		ChartGrid: "#3e3e3e",

		SearchHighlight: "#f9e2af",
		HelpKey:         "#7C3AED",
		HelpDesc:        "#6b6b6b",
	}
}

// thGruvboxTheme returns the warm retro Gruvbox theme.
func thGruvboxTheme() Theme {
	return Theme{
		Name:       "gruvbox",
		Background: "#282828",
		Foreground: "#ebdbb2",
		Dim:        "#928374",
		Accent:     "#fe8019",

		Border:      "#504945",
		BorderFocus: "#fe8019",
		Title:       "#ebdbb2",

		StatusOK:      "#b8bb26",
		StatusWarn:    "#fabd2f",
		StatusError:   "#fb4934",
		StatusUnknown: "#928374",

		GaugeFilled: "#b8bb26",
		GaugeEmpty:  "#504945",
		GaugeWarn:   "#fabd2f",
		GaugeCrit:   "#fb4934",

		ChartLine: "#fe8019",
		ChartFill: "#d65d0e",
		ChartGrid: "#504945",

		SearchHighlight: "#fabd2f",
		HelpKey:         "#fe8019",
		HelpDesc:        "#928374",
	}
}

// thNordTheme returns the arctic blue Nord theme.
func thNordTheme() Theme {
	return Theme{
		Name:       "nord",
		Background: "#2e3440",
		Foreground: "#eceff4",
		Dim:        "#4c566a",
		Accent:     "#88c0d0",

		Border:      "#3b4252",
		BorderFocus: "#88c0d0",
		Title:       "#eceff4",

		StatusOK:      "#a3be8c",
		StatusWarn:    "#ebcb8b",
		StatusError:   "#bf616a",
		StatusUnknown: "#4c566a",

		GaugeFilled: "#a3be8c",
		GaugeEmpty:  "#3b4252",
		GaugeWarn:   "#ebcb8b",
		GaugeCrit:   "#bf616a",

		ChartLine: "#88c0d0",
		ChartFill: "#5e81ac",
		ChartGrid: "#3b4252",

		SearchHighlight: "#ebcb8b",
		HelpKey:         "#88c0d0",
		HelpDesc:        "#4c566a",
	}
}

// thCatppuccinTheme returns the pastel Catppuccin Mocha theme.
func thCatppuccinTheme() Theme {
	return Theme{
		Name:       "catppuccin",
		Background: "#1e1e2e",
		Foreground: "#cdd6f4",
		Dim:        "#6c7086",
		Accent:     "#cba6f7",

		Border:      "#313244",
		BorderFocus: "#cba6f7",
		Title:       "#cdd6f4",

		StatusOK:      "#a6e3a1",
		StatusWarn:    "#f9e2af",
		StatusError:   "#f38ba8",
		StatusUnknown: "#6c7086",

		GaugeFilled: "#a6e3a1",
		GaugeEmpty:  "#313244",
		GaugeWarn:   "#f9e2af",
		GaugeCrit:   "#f38ba8",

		ChartLine: "#cba6f7",
		ChartFill: "#9399b2",
		ChartGrid: "#313244",

		SearchHighlight: "#f9e2af",
		HelpKey:         "#cba6f7",
		HelpDesc:        "#6c7086",
	}
}

// thDraculaTheme returns the Dracula theme.
func thDraculaTheme() Theme {
	return Theme{
		Name:       "dracula",
		Background: "#282a36",
		Foreground: "#f8f8f2",
		Dim:        "#6272a4",
		Accent:     "#bd93f9",

		Border:      "#44475a",
		BorderFocus: "#bd93f9",
		Title:       "#f8f8f2",

		StatusOK:      "#50fa7b",
		StatusWarn:    "#f1fa8c",
		StatusError:   "#ff5555",
		StatusUnknown: "#6272a4",

		GaugeFilled: "#50fa7b",
		GaugeEmpty:  "#44475a",
		GaugeWarn:   "#f1fa8c",
		GaugeCrit:   "#ff5555",

		ChartLine: "#bd93f9",
		ChartFill: "#8be9fd",
		ChartGrid: "#44475a",

		SearchHighlight: "#f1fa8c",
		HelpKey:         "#bd93f9",
		HelpDesc:        "#6272a4",
	}
}

// thTokyoNightTheme returns the Tokyo Night theme.
func thTokyoNightTheme() Theme {
	return Theme{
		Name:       "tokyo-night",
		Background: "#1a1b26",
		Foreground: "#c0caf5",
		Dim:        "#565f89",
		Accent:     "#7aa2f7",

		Border:      "#292e42",
		BorderFocus: "#7aa2f7",
		Title:       "#c0caf5",

		StatusOK:      "#9ece6a",
		StatusWarn:    "#e0af68",
		StatusError:   "#f7768e",
		StatusUnknown: "#565f89",

		GaugeFilled: "#9ece6a",
		GaugeEmpty:  "#292e42",
		GaugeWarn:   "#e0af68",
		GaugeCrit:   "#f7768e",

		ChartLine: "#7aa2f7",
		ChartFill: "#7dcfff",
		ChartGrid: "#292e42",

		SearchHighlight: "#e0af68",
		HelpKey:         "#7aa2f7",
		HelpDesc:        "#565f89",
	}
}
