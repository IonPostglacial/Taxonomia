package dataset

import (
	"encoding/json"
	"io"
)

var extraneousProps = []string{
	"vernacularName2",
	"name2",
	"herbariumPicture",
	"website",
	"noHerbier",
	"fasc",
	"page",
}

func encodePictures(pics []Picture) []EncodedPhoto {
	photos := make([]EncodedPhoto, len(pics))
	for i, pic := range pics {
		photos[i] = EncodedPhoto{
			Id:    pic.Id,
			Url:   pic.Source,
			Label: pic.Legend,
		}
	}
	return photos
}

func encodeItem(hierarchy *Hierarchy, parentId string, childrenIds []string) EncodedItem {
	return EncodedItem{
		Id:             hierarchy.Id,
		ParentId:       parentId,
		Name:           hierarchy.Name.Scientific,
		NameCN:         hierarchy.Name.Text("CN"),
		NameEN:         hierarchy.Name.Text("EN"),
		VernacularName: hierarchy.Name.Text("NV"),
		Photos:         encodePictures(hierarchy.Pictures),
		Children:       childrenIds,
	}
}

func encodeTaxon(ds *Dataset, taxon *Taxon, parentId string, charByStateId map[string]string, out *[]*EncodedTaxon) {
	statesByChar := map[string][]string{}
	for _, state := range taxon.States {
		charId := charByStateId[state.Id]
		statesByChar[charId] = append(statesByChar[charId], state.Id)
	}
	descriptions := make([]EncodedDescriptions, 0, len(statesByChar))
	for charId, stateIds := range statesByChar {
		descriptions = append(descriptions, EncodedDescriptions{
			DescriptorId: charId,
			StatesIds:    stateIds,
		})
	}
	extras := map[string]interface{}{}
	for k, v := range taxon.ExtraInfo {
		extras[k] = v
	}
	for _, extraProp := range extraneousProps {
		delete(extras, extraProp)
	}
	bookInfoByIds := map[string]EncodedBookInfo{}
	childrenIds := make([]string, len(taxon.Children))
	for i, child := range taxon.Children {
		childrenIds[i] = child.Id
	}
	*out = append(*out, &EncodedTaxon{
		EncodedItem:   encodeItem(taxon.Hierarchy, parentId, childrenIds),
		Author:        taxon.Author,
		Descriptions:  descriptions,
		BookInfoByIds: bookInfoByIds,
		Extra:         extras,
	})
	for _, ch := range taxon.Children {
		child := ds.TaxonsById[ch.Id]
		encodeTaxon(ds, child, taxon.Id, charByStateId, out)
	}
}

func encodeCharacter(ds *Dataset, ch *Character, parentId string, stateIds []string, out *[]*EncodedCharacter) {
	reqIds := make([]string, len(ch.RequiredStates))
	for i, state := range ch.RequiredStates {
		reqIds[i] = state.Id
	}
	inappIds := make([]string, len(ch.InapplicableStates))
	for i, state := range ch.InapplicableStates {
		inappIds[i] = state.Id
	}
	var inherentStateId string
	if ch.InherentState != nil {
		inherentStateId = ch.InherentState.Id
	}
	childrenIds := make([]string, len(ch.Children))
	for i, child := range ch.Children {
		childrenIds[i] = child.Id
	}
	*out = append(*out, &EncodedCharacter{
		EncodedItem:           encodeItem(ch.Hierarchy, parentId, childrenIds),
		InherentStateId:       inherentStateId,
		States:                stateIds,
		RequiredStatesIds:     reqIds,
		InapplicableStatesIds: inappIds,
	})
	for _, h := range ch.Children {
		if child, ok := ds.CharactersById[h.Id]; ok {
			encodeCharacter(ds, child, ch.Id, stateIds, out)
		}
	}
}

func encodeState(state *State) *EncodedState {
	return &EncodedState{
		Id:          state.Id,
		Name:        state.Name.Scientific,
		NameEN:      state.Name.Text("EN"),
		NameCN:      state.Name.Text("CN"),
		Photos:      encodePictures(state.Pictures),
		Description: state.Description,
		Color:       state.Color,
	}
}

func WriteHazo(w io.Writer, dataset *Dataset) error {
	encoded := Encoded{
		Id:                dataset.Id,
		Taxons:            []*EncodedTaxon{},
		Characters:        []*EncodedCharacter{},
		States:            []*EncodedState{},
		Books:             []*EncodedBook{},
		ExtraFields:       []*EncodedExtraField{},
		DictionaryEntries: make(map[string]*EncodedDictionaryEntry),
	}
	charByStateId := map[string]string{}
	for _, character := range dataset.CharactersById {
		charStateIds := make([]string, len(character.States))
		for i, state := range character.States {
			encodedState := encodeState(&state)
			charStateIds[i] = encodedState.Id
			encoded.States = append(encoded.States, encodedState)
			charByStateId[state.Id] = character.Id
		}
		encodeCharacter(dataset, character, "", charStateIds, &encoded.Characters)
	}
	for _, taxon := range dataset.TaxonsById {
		encodeTaxon(dataset, taxon, "", charByStateId, &encoded.Taxons)
	}
	result, err := json.Marshal(&encoded)
	if err != nil {
		return err
	}
	w.Write(result)
	return nil
}
