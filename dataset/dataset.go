package dataset

import "fmt"

type Dataset struct {
	Id                  string
	TaxonsHierarchy     *Hierarchy
	CharactersHierarchy *Hierarchy
	TaxonsById          map[string]*Taxon
	CharactersById      map[string]*Character
	DictionaryEntry     []DictionaryEntry
	ExtraFields         []ExtraField
}

func New(id string) *Dataset {
	return &Dataset{
		Id: id,
		TaxonsHierarchy: &Hierarchy{
			Id: "t0",
			Name: MultilangText{
				Scientific:     "Taxons",
				NamesByLangRef: map[string]string{},
			},
		},
		CharactersHierarchy: &Hierarchy{
			Id: "c0",
			Name: MultilangText{
				Scientific:     "Characters",
				NamesByLangRef: map[string]string{},
			},
		},
		TaxonsById:     map[string]*Taxon{},
		CharactersById: map[string]*Character{},
	}
}

func generateNewId(prefix string, size int, has func(id string) bool) string {
	newIndex := size + 1
	newId := fmt.Sprintf("%s%d", prefix, newIndex)
	ko := has(newId)
	for ; ko; newIndex++ {
		newId = fmt.Sprintf("%s%d", prefix, newIndex)
		ko = has(newId)
	}
	return newId
}

type TaxonInit struct {
	Id          string
	Name        MultilangText
	Description string
	Author      string
	States      []*State
	References  []BookReference
	ExtraInfo   map[string]interface{}
}

func (ds *Dataset) AddTaxonBelow(taxon *Taxon, parent *Hierarchy) {
	ds.TaxonsById[taxon.Id] = taxon
	if parent == nil {
		ds.TaxonsHierarchy.Children = append(ds.TaxonsHierarchy.Children, taxon.Hierarchy)
	} else {
		parent.Children = append(parent.Children, taxon.Hierarchy)
	}
}

func (ds *Dataset) CreateTaxon(path []int, init TaxonInit) []int {
	newId := init.Id
	if newId == "" {
		newId = generateNewId("t", len(ds.TaxonsById), func(id string) bool {
			_, ok := ds.TaxonsById[id]
			return ok
		})
	}
	hierarchy := &Hierarchy{Id: newId, Name: init.Name, Description: init.Description}
	extraInfo := init.ExtraInfo
	if extraInfo == nil {
		extraInfo = map[string]interface{}{}
	}
	taxon := &Taxon{
		Hierarchy:  hierarchy,
		Author:     init.Author,
		States:     init.States,
		References: init.References,
		ExtraInfo:  extraInfo,
	}
	parent := ds.TaxonsHierarchy.GetIn(path)
	ds.AddTaxonBelow(taxon, parent)
	return append(path, len(parent.Children))
}

func (ds *Dataset) AddCharacterBelow(ch *Character, parent *Hierarchy) {
	ds.CharactersById[ch.Id] = ch
	if parent == nil {
		ds.TaxonsHierarchy.Children = append(ds.TaxonsHierarchy.Children, ch.Hierarchy)
	} else {
		parent.Children = append(parent.Children, ch.Hierarchy)
	}
}

func (ds *Dataset) CreateCharacter(path []int, name *MultilangText) {
	newId := generateNewId("c", len(ds.CharactersById), func(id string) bool {
		_, ok := ds.CharactersById[id]
		return ok
	})
	hierarchy := &Hierarchy{Id: newId, Name: *name}
	parent := ds.CharactersHierarchy.GetIn(path)
	ch := NewCharacter(hierarchy)
	ds.AddCharacterBelow(ch, parent)
}
