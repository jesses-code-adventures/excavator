package core

import (
	"github.com/charmbracelet/bubbles/textinput"
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
	return [...]string{"DirectoryWalker", "FormWindow", "ListSelectionWindow", "SearchableSelectableList"}[w]
}

// A singular form input control
type FormInput struct {
	Name  string
	Input textinput.Model
}

// Constructor for a form input
func NewFormInput(name string) FormInput {
	return FormInput{
		Name:  name,
		Input: textinput.New(),
	}
}

// A generic Form
type Form struct {
	Title        string
	Inputs       []FormInput
	Writing      bool
	FocusedInput int
}

// A form constructor
func NewForm(title string, inputs []FormInput) Form {
	return Form{
		Title:        title,
		Inputs:       inputs,
		Writing:      false,
		FocusedInput: 0,
	}
}

//// Form implementations ////

// Get the inputs for the new collection form
func GetNewCollectionInputs() []FormInput {
	return []FormInput{
		NewFormInput("name"),
		NewFormInput("description"),
	}
}

// Get the new collection form
func GetNewCollectionForm() Form {
	return NewForm("create collection", GetNewCollectionInputs())
}

// Get the inputs for the new collection form
func GetSearchInput() []FormInput {
	return []FormInput{
		NewFormInput("search"),
	}
}

// Get the new collection form
func GetTargetSubCollectionForm() Form {
	return NewForm("set target subcollection", GetSearchInput())
}

// Get the new collection form
func GetFuzzySearchRootForm() Form {
	return NewForm("fuzzy search from root", GetSearchInput())
}

// Get the inputs for the new collection form
func GetCreateCollectionTagInputs(defaultName string, defaultSubCollection string) []FormInput {
	name := NewFormInput("name")
	name.Input.SetValue(defaultName)
	subcollection := NewFormInput("subcollection")
	subcollection.Input.SetValue(defaultSubCollection)
	return []FormInput{
		name,
		subcollection,
	}
}

// Get the new collection form
func GetCreateTagForm(defaultName string, defaultSubCollection string) Form {
	return NewForm("create tag", GetCreateCollectionTagInputs(defaultName, defaultSubCollection))
}

/// List selection ///

// Interface for list selection items so the list can easily be reused
type SelectableListItem interface {
	Id() int
	Name() string
	Description() string
	IsDir() bool
	IsFile() bool
}

// A list where a single item can be selected
type SelectableList struct {
	Title string
}

// A constructor for a selectable list
type SearchableSelectableList struct {
	Title  string
	Search FormInput
}

func NewSearchableList(title string) SearchableSelectableList {
	return SearchableSelectableList{
		Title: title,
		Search: FormInput{
			Name:  "search",
			Input: textinput.New(),
		},
	}
}
