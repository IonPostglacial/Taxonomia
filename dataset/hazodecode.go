package dataset

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"strconv"
)

func stringOrArrayQuirck(stringOrArray interface{}) string {
	src := ""
	switch url := stringOrArray.(type) {
	case string:
		src = url
	case []string:
		if len(url) > 0 {
			src = url[0]
		}
	}
	return src
}

func decodePictures(photos []EncodedPhoto) []Picture {
	pics := make([]Picture, len(photos))
	for i, photo := range photos {
		pics[i] = Picture{
			Id:     photo.Id,
			Source: stringOrArrayQuirck(photo.Url),
			Legend: stringOrArrayQuirck(photo.Label),
		}
	}
	return pics
}

type charNode struct {
	Entry    *Character
	Children []*Character
}

func decodeHierarchy(encoded *EncodedItem) *Hierarchy {
	name := MultilangText{Scientific: encoded.Name, NamesByLangRef: map[string]string{}}
	if encoded.NameEN != "" {
		name.NamesByLangRef["EN"] = encoded.NameEN
	}
	if encoded.NameCN != "" {
		name.NamesByLangRef["CN"] = encoded.NameCN
	}
	return &Hierarchy{
		Id:          encoded.Id,
		Name:        name,
		Description: encoded.Detail,
	}
}

func decodeStatesByIds(states []*EncodedState) map[string]*State {
	statesByIds := make(map[string]*State)
	for _, state := range states {
		statesByIds[state.Id] = &State{
			Id:          state.Id,
			Description: state.Description,
			Name: MultilangText{
				Scientific: state.Name,
				NamesByLangRef: map[string]string{
					"CN": state.NameCN,
					"EN": state.NameEN,
					"FR": state.Name,
				},
			},
			Pictures: decodePictures(state.Photos),
			Color:    state.Color,
		}
	}
	return statesByIds
}

func decodeTaxonsByIds(encodedTaxons []*EncodedTaxon, statesByIds map[string]*State) map[string]*Taxon {
	taxonsByIds := map[string]*Taxon{}
	for _, taxon := range encodedTaxons {
		states := make([]*State, 0)
		for _, desc := range taxon.Descriptions {
			for _, stateId := range desc.StatesIds {
				state, ok := statesByIds[stateId]
				if ok {
					states = append(states, state)
				}
			}
		}
		refs := make([]BookReference, 0, len(taxon.BookInfoByIds))
		for _, bookInfo := range taxon.BookInfoByIds {
			page, err := strconv.Atoi(bookInfo.Page)
			if err != nil {
				page = 0
			}
			refs = append(refs, BookReference{
				Page:   page,
				Fasc:   bookInfo.Fasc,
				Detail: bookInfo.Detail,
			})
		}
		extras := map[string]interface{}{}
		for k, v := range taxon.Extra {
			extras[k] = v
		}
		addExtra := func(key string, value interface{}) {
			if value != "" && value != 0 {
				extras[key] = value
			}
		}
		addExtra("vernacularName2", taxon.VernacularName2)
		addExtra("name2", taxon.Name2)
		addExtra("herbariumpicture", taxon.HerbariumPicture)
		addExtra("website", taxon.Website)
		addExtra("noHerbier", taxon.NoHerbier)
		addExtra("fasc", taxon.Fasc)
		addExtra("page", taxon.Page)
		hierarchy := decodeHierarchy(&taxon.EncodedItem)
		taxon := &Taxon{
			Hierarchy:  hierarchy,
			Author:     taxon.Author,
			States:     states,
			References: refs,
			ExtraInfo:  extras,
		}
		taxonsByIds[taxon.Id] = taxon
	}
	return taxonsByIds
}

func decodeCharactersByIds(encodedCharacters []*EncodedCharacter, statesByIds map[string]*State) map[string]*Character {
	charactersByIds := map[string]*Character{}
	for _, ch := range encodedCharacters {
		states := make([]State, len(ch.States))
		for i, stateId := range ch.States {
			states[i] = *statesByIds[stateId]
		}
		inapplicableStates := make([]*State, len(ch.InapplicableStatesIds))
		for i, stateId := range ch.InapplicableStatesIds {
			inapplicableStates[i] = statesByIds[stateId]
		}
		requiredStates := make([]*State, len(ch.RequiredStatesIds))
		for i, stateId := range ch.RequiredStatesIds {
			requiredStates[i] = statesByIds[stateId]
		}
		character := &Character{
			Hierarchy:          decodeHierarchy(&ch.EncodedItem),
			InherentState:      statesByIds[ch.InherentStateId],
			States:             states,
			InapplicableStates: inapplicableStates,
			RequiredStates:     requiredStates,
		}
		charactersByIds[character.Id] = character
	}
	return charactersByIds
}

func ReadHazo(r io.Reader) (*Dataset, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	encodedDataset := Encoded{}
	err = json.Unmarshal(data, &encodedDataset)
	if err != nil {
		return nil, err
	}
	dataset := New(encodedDataset.Id)
	statesByIds := decodeStatesByIds(encodedDataset.States)
	dataset.TaxonsById = decodeTaxonsByIds(encodedDataset.Taxons, statesByIds)
	dataset.CharactersById = decodeCharactersByIds(encodedDataset.Characters, statesByIds)
	for _, t := range encodedDataset.Taxons {
		taxon := dataset.TaxonsById[t.Id]
		parent := dataset.TaxonsById[t.ParentId]
		if taxon != nil {
			if parent != nil {
				dataset.AddTaxonBelow(taxon, parent.Hierarchy)
			} else {
				dataset.AddTaxonBelow(taxon, dataset.TaxonsHierarchy)
			}
		}
	}
	for _, ch := range encodedDataset.Characters {
		character := dataset.CharactersById[ch.Id]
		parent := dataset.CharactersById[ch.ParentId]
		if parent != nil {
			dataset.AddCharacterBelow(character, parent.Hierarchy)
		} else {
			dataset.AddCharacterBelow(character, dataset.CharactersHierarchy)
		}
	}
	return dataset, nil
}
