package window

import (
	"github.com/charmbracelet/lipgloss"
)

// ////////////////////// STYLING ////////////////////////
// All styles to be used throughout the ui
var (
	// colours
	Green = lipgloss.Color("#25A065")
	Pink  = lipgloss.Color("#E441B5")
	White = lipgloss.Color("#FFFDF5")
	// App
	AppStyle   = lipgloss.NewStyle()
	TitleStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Green).
			Padding(1, 1).
			Height(3)
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#909090",
			Dark:  "#626262",
		})
	HelpValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{
			Light: "#B2B2B2",
			Dark:  "#4A4A4A",
		})
	// Directory Walker
	ViewportStyle = lipgloss.NewStyle()
	SelectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder()).
			Foreground(Pink)
	UnselectedStyle = lipgloss.NewStyle().
			Border(lipgloss.HiddenBorder())
		// Form
	FormStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Border(lipgloss.HiddenBorder())
	UnfocusedInput = lipgloss.NewStyle().
			Width(100).
			Border(lipgloss.HiddenBorder())
	FocusedInput = lipgloss.NewStyle().
			Width(100).
			Foreground(Pink).
			Border(lipgloss.HiddenBorder())
		// Searchable list
	SearchableListItemsStyle = lipgloss.NewStyle().
					Border(lipgloss.HiddenBorder())
	SearchInputBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()).
				AlignVertical(lipgloss.Bottom).
				AlignHorizontal(lipgloss.Left)
	SearchableSelectableListStyle = lipgloss.NewStyle().
					Border(lipgloss.HiddenBorder())
	PreViewportPromptStyle = lipgloss.NewStyle().
				Border(lipgloss.HiddenBorder()).
				Foreground(Pink)
	PreViewportInputStyle = lipgloss.NewStyle().
				Width(100).
				Foreground(Pink).
				Border(lipgloss.HiddenBorder())
	PreViewportStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Border(lipgloss.HiddenBorder())
)
