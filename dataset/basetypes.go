package dataset

type Lang struct {
	Code  string
	Label string
}

type MultilangText struct {
	Scientific     string
	NamesByLangRef map[string]string
}

func NewMultilangText(sciName string) *MultilangText {
	return &MultilangText{Scientific: sciName, NamesByLangRef: map[string]string{}}
}

func (name MultilangText) Text(langRef string) string {
	if text, ok := name.NamesByLangRef[langRef]; ok {
		return text
	} else {
		return name.Scientific
	}
}

func (text MultilangText) Equals(other MultilangText) bool {
	if text.Scientific != other.Scientific || len(text.NamesByLangRef) != len(other.NamesByLangRef) {
		return false
	} else if text.NamesByLangRef == nil {
		return other.NamesByLangRef == nil || len(other.NamesByLangRef) == 0
	} else if other.NamesByLangRef == nil {
		return len(text.NamesByLangRef) == 0
	}
	for lang, name := range text.NamesByLangRef {
		if name != other.NamesByLangRef[lang] {
			return false
		}
	}
	return true
}

type Picture struct {
	Id     string
	Source string
	Legend string
}

type Hierarchy struct {
	Id          string
	Name        MultilangText
	Description string
	Pictures    []Picture
	Children    []*Hierarchy
}

func (h *Hierarchy) GetIn(path []int) *Hierarchy {
	it := h
	for _, index := range path {
		it = it.Children[index]
	}
	return it
}

type State struct {
	Id          string
	Name        MultilangText
	Pictures    []Picture
	Description string
	Color       string
}

type BookReference struct {
	Page   int
	Fasc   string
	Detail string
}

type Taxon struct {
	*Hierarchy
	Author     string
	States     []*State
	References []BookReference
	ExtraInfo  map[string]interface{}
}

func NewTaxon(hierarchy *Hierarchy) *Taxon {
	return &Taxon{
		Hierarchy: hierarchy,
		ExtraInfo: map[string]interface{}{},
	}
}

type Character struct {
	*Hierarchy
	InherentState      *State
	States             []State
	InapplicableStates []*State
	RequiredStates     []*State
}

func NewCharacter(hierarchy *Hierarchy) *Character {
	return &Character{Hierarchy: hierarchy}
}

type DictionaryEntry struct {
	Id         string
	Url        string
	Name       MultilangText
	Definition MultilangText
}

type ExtraField struct {
	IsStandard bool
	Id         string
	Label      string
	Icon       string
}

type Book struct {
	Id    string
	Title string
}
