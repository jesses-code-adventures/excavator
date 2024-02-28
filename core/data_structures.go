package core

// Different window types
type WindowType int

const (
	DirectoryWalker WindowType = iota
	FormWindow
	ListSelectionWindow
	SearchableSelectableListWindow
)

func (w WindowType) String() string {
	return [...]string{"DirectoryWalker", "FormWindow", "ListSelectionWindow", "SearchableSelectableList"}[w]
}

