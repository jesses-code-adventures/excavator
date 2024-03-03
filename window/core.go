package window

import (
	"fmt"
	"log"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/jesses-code-adventures/excavator/core"
)

// Different window types
type WindowType int

const (
	DirectoryWalker WindowType = iota
	FormWindow
	ListSelectionWindow
	SearchableSelectableListWindow
)

func (w WindowType) String() string {
	return [...]string{"DirectoryWalker", "FormWindow", "ListSelectionWindow", "SearchableSelectableListWindow"}[w]
}

type WindowName int

const (
	Home WindowName = iota
	NewCollectionWindow
	NewTagWindow
	SetTargetSubCollectionWindow
	SetTargetCollectionWindow
	FuzzySearchRootWindow
	FuzzySearchCurrentWindow
	CreateExportWindow
	RunExportWindow
)

func (w WindowName) String() string {
	return [...]string{"home window", "new collection", "create tag", "set target subcollection", "set target collection", "fuzzy search from root", "fuzzy search window", "create export", "run export"}[w]
}

func (w WindowName) Window() Window {
	switch w {
	case Home:
		return Window{
			name:       w,
			windowType: DirectoryWalker,
		}
	case NewCollectionWindow:
		return Window{
			name:       w,
			windowType: FormWindow,
		}
	case NewTagWindow:
		return Window{
			name:       w,
			windowType: FormWindow,
		}
	case SetTargetSubCollectionWindow:
		return Window{
			name:       w,
			windowType: SearchableSelectableListWindow,
		}
	case SetTargetCollectionWindow:
		log.Println("SetTargetCollectionWindow")
		return Window{
			name:       w,
			windowType: ListSelectionWindow,
		}
	case FuzzySearchRootWindow:
		return Window{
			name:       w,
			windowType: SearchableSelectableListWindow,
		}
	case FuzzySearchCurrentWindow:
		return Window{
			name:       w,
			windowType: SearchableSelectableListWindow,
		}
	case CreateExportWindow:
		return Window{
			name:       w,
			windowType: FormWindow,
		}
	case RunExportWindow:
		return Window{
			name:       w,
			windowType: ListSelectionWindow,
		}
	default:
		log.Fatalf("Unknown window name: %v", w.String())
	}
	return Window{}
}

func (w WindowType) FormView(form core.Form) string {
	if w != FormWindow {
		log.Fatalf("FormView called on a non-form window: %v", w)
	}
	log.Println("got form ", form.Title)
	s := ""
	for i, input := range form.Inputs {
		if form.FocusedInput == i {
			s += FocusedInput.Render(fmt.Sprintf("%v: %v", input.Name, input.Input.View()))
		} else {
			s += UnfocusedInput.Render(fmt.Sprintf("%v: %v", input.Name, input.Input.View()))
		}
	}
	return FormStyle.Render(s)
}

func (w WindowType) DirectoryView(choices []core.SelectableListItem, cursor int, maxWidth int, showCollections bool, input textinput.Model) string {
	if w == FormWindow {
		log.Fatal("Directory view called on a form", w)
	}
	s := ""
	for i, choice := range choices {
		var newLine string
		if cursor == i {
			cursor := ">"
			newLine = fmt.Sprintf("%s %s", cursor, choice.Name())
		} else {
			newLine = fmt.Sprintf("  %s", choice.Name())
		}
		if len(newLine) > maxWidth {
			newLine = newLine[:maxWidth-2]
		}
		if cursor == i {
			newLine = SelectedStyle.Render(newLine, fmt.Sprintf("    %v", choice.Description()))
		} else {
			if showCollections {
				newLine = UnselectedStyle.Render(newLine, fmt.Sprintf("    %v", choice.Description()))
			} else {
				newLine = UnselectedStyle.Render(newLine)
			}
		}
		s += newLine
	}
	var searchInput string
	if w == SearchableSelectableListWindow {
		// searchInput = SearchInputBoxStyle.Render(m.SearchableSelectableList.Search.Input.View())
		searchInput = SearchInputBoxStyle.Render(input.View())
		return s + "\n" + searchInput
	}
	return s
}

type Window struct {
	name       WindowName
	windowType WindowType
}

func (w Window) Name() WindowName {
	return w.name
}

func (w Window) Type() WindowType {
	return w.windowType
}
