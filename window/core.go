package window

import (
	"fmt"
	"log"
	"path"

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
	PreViewport
)

func (w WindowType) String() string {
	return [...]string{"DirectoryWalker", "FormWindow", "ListSelectionWindow", "SearchableSelectableListWindow", "PreViewport"}[w]
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
	BrowseCollectionWindow
	EnterUserWindow
	EnterRootWindow
)

func (w WindowName) String() string {
	return [...]string{"home", "create collection", "create tag", "target subcollection", "target collection", "recursive search - root", "recursive search - current dir", "create export", "run export", "browse target collection", "create user", "create root"}[w]
}

func (w WindowName) Window() Window {
	switch w {
	case Home:
		return Window{
			name:       w,
			windowType: SearchableSelectableListWindow,
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
	case BrowseCollectionWindow:
		return Window{
			name:       w,
			windowType: SearchableSelectableListWindow,
		}
	case EnterUserWindow:
		return Window{
			name:       w,
			windowType: PreViewport,
		}
	case EnterRootWindow:
		return Window{
			name:       w,
			windowType: PreViewport,
		}
	default:
		log.Fatalf("Unknown window name: %v", w.String())
	}
	return Window{}
}

func (w WindowType) PreViewportView(prompt string, input textinput.Model) string {
	if w != PreViewport {
		log.Fatalf("FormView called on a non-form window: %v", w)
	}
	s := ""
	prompt = PreViewportPromptStyle.Render(prompt)
	s += prompt
	renderedInput := PreViewportInputStyle.Render(input.View())
	s += renderedInput
	return PreViewportStyle.Render(s)
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

func (w WindowType) SearchableListView(choices []core.SelectableListItem, cursor int, maxWidth int, showCollections bool, input textinput.Model, isBrowseCollectionView bool) string {
	if w == FormWindow {
		log.Fatal("Searchable list view called on a form", w)
	}
	s := ""
	for i, choice := range choices {
		var newLine string
		var name string
		var description string
		if isBrowseCollectionView {
			name = path.Join(choice.Description(), choice.Name())
			description = ""
		} else {
			name = choice.Name()
			description = choice.Description()
		}
		if cursor == i {
			cursor := ">"
			newLine = fmt.Sprintf("%s %s", cursor, name)
		} else {
			newLine = fmt.Sprintf("  %s", name)
		}
		if len(newLine) > maxWidth {
			newLine = newLine[:maxWidth-2]
		}
		if cursor == i {
			newLine = SelectedStyle.Render(newLine, fmt.Sprintf("    %v", description))
		} else {
			if showCollections {
				newLine = UnselectedStyle.Render(newLine, fmt.Sprintf("    %v", description))
			} else {
				newLine = UnselectedStyle.Render(newLine)
			}
		}
		s += newLine
	}
	var searchInput string
	if w == SearchableSelectableListWindow {
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
