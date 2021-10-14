package dataset

type EncodedPhoto struct {
	Id    string      `json:"id"`
	Url   interface{} `json:"url"`
	Label interface{} `json:"label"`
}

type EncodedState struct {
	Id          string         `json:"id"`
	Name        string         `json:"name"`
	NameEN      string         `json:"nameEN"`
	NameCN      string         `json:"nameCN"`
	Photos      []EncodedPhoto `json:"photos"`
	Description string         `json:"description"`
	Color       string         `json:"color,omitempty"`
}

type EncodedBook struct {
	Id    string `json:"id"`
	Label string `json:"label"`
}

type EncodedExtraField struct {
	Std   bool   `json:"std"`
	Id    string `json:"id"`
	Label string `json:"label"`
	Icon  string `json:"icon"`
}

type EncodedDictionaryEntry struct {
	Id     int    `json:"id"`
	NameCN string `json:"nameCN"`
	NameEN string `json:"nameEN"`
	NameFR string `json:"nameFR"`
	DefCN  string `json:"defCN"`
	DefEN  string `json:"defEN"`
	DefFR  string `json:"defFR"`
	Url    string `json:"url"`
}

type EncodedItem struct {
	Id             string         `json:"id"`
	ParentId       string         `json:"parentId,omitempty"`
	Name           string         `json:"name"`
	NameEN         string         `json:"nameEN"`
	NameCN         string         `json:"nameCN"`
	VernacularName string         `json:"vernacularName"`
	Detail         string         `json:"detail"`
	Children       []string       `json:"children"`
	Photos         []EncodedPhoto `json:"photos"`
}

type EncodedBookInfo struct {
	Fasc   string `json:"fasc"`
	Page   string `json:"page"`
	Detail string `json:"detail"`
}

type EncodedDescriptions struct {
	DescriptorId string   `json:"descriptorId"`
	StatesIds    []string `json:"statesIds"`
}

type EncodedTaxon struct {
	EncodedItem
	Descriptions     []EncodedDescriptions      `json:"descriptions"`
	Author           string                     `json:"author"`
	VernacularName2  string                     `json:"vernacularName2,omitempty"`
	Name2            string                     `json:"name2,omitempty"`
	Meaning          string                     `json:"meaning,omitempty"`
	HerbariumPicture string                     `json:"herbariumpicture,omitempty"`
	Website          string                     `json:"website,omitempty"`
	NoHerbier        string                     `json:"noHerbier,omitempty"`
	Fasc             string                     `json:"fasc,omitempty"`
	Page             string                     `json:"page,omitempty"`
	BookInfoByIds    map[string]EncodedBookInfo `json:"bookInfobyids,omitempty"`
	Extra            map[string]interface{}     `json:"extra,omitempty"`
}

type EncodedCharacter struct {
	EncodedItem
	InherentStateId       string   `json:"inherentstateid"`
	States                []string `json:"states"`
	RequiredStatesIds     []string `json:"requiredStatesIds"`
	InapplicableStatesIds []string `json:"inapplicablestatesids"`
}

type Encoded struct {
	Id                string                             `json:"id"`
	Taxons            []*EncodedTaxon                    `json:"taxons"`
	Characters        []*EncodedCharacter                `json:"characters"`
	States            []*EncodedState                    `json:"states"`
	Books             []*EncodedBook                     `json:"books"`
	ExtraFields       []*EncodedExtraField               `json:"extraFields"`
	DictionaryEntries map[string]*EncodedDictionaryEntry `json:"dictionaryEntries"`
}
