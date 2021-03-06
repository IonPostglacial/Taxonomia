package database

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"nicolas.galipot.net/taxonomia/dataset"
)

func CreateTables(db *sql.DB) (err error) {
	sqlTables := []string{"Items", "PictureCache", "ItemPictures", "Languages", "ItemNames", "Hierarchies", "Characters", "States", "Taxons", "TaxonStates", "CharacterRequiredStates"}
	sqlCreateTables := []string{
		`CREATE TABLE Items (
			id TEXT NOT NULL,
			ord INTEGER NOT NULL,
			name VARCHAR(512) NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (id)
		);`,
		`CREATE TABLE PictureCache (
			src TEXT NOT NULL,
			data BLOB NOT NULL,
			PRIMARY KEY (src)
		);`,
		`CREATE TABLE ItemPictures (
			id INT NOT NULL,
			item TEXT NOT NULL,
			url VARCHAR(512) NOT NULL,
			label VARCHAR(512) NOT NULL,
			PRIMARY KEY (id),
			FOREIGN KEY(item) REFERENCES Items(id)
		);`,
		`CREATE TABLE Languages (
			code VARCHAR(2) NOT NULL,
			label VARCHAR(512) NOT NULL,
			PRIMARY KEY (code)
		);`,
		`CREATE TABLE ItemNames (
			item TEXT NOT NULL,
			lang VARCHAR(2) NOT NULL,
			text VARCHAR(512) NOT NULL,
			PRIMARY KEY(item, lang),
			FOREIGN KEY(item) REFERENCES Items(id),
			FOREIGN KEY(lang) REFERENCES Languages(id)
		);`,
		`CREATE TABLE Hierarchies (
			ancestor TEXT NOT NULL,
			descendant TEXT NOT NULL,
			length INT NOT NULL DEFAULT 0,
			PRIMARY KEY(ancestor, descendant),
			FOREIGN KEY(ancestor) REFERENCES Items(id),
			FOREIGN KEY(descendant) REFERENCES Items(id)
		);`,
		`CREATE TABLE Characters (
			item TEXT NOT NULL,
			PRIMARY KEY(item),
			FOREIGN KEY(item) REFERENCES Items(id)
		);`,
		`CREATE TABLE States (
			item TEXT NOT NULL,
			character INT NOT NULL,
			color INT NOT NULL DEFAULT 0,
			PRIMARY KEY(item),
			FOREIGN KEY(item) REFERENCES Items(id),
			FOREIGN KEY(character) REFERENCES Characters(id)
		);`,
		`CREATE TABLE Taxons (
			item TEXT NOT NULL,
			author VARCHAR(512) NOT NULL,
			PRIMARY KEY(item)
		);`,
		`CREATE TABLE TaxonStates (
			taxon TEXT NOT NULL,
			state TEXT NOT NULL,
			PRIMARY KEY(taxon, state)
		);`,
		`CREATE TABLE CharacterRequiredStates (
			character TEXT NOT NULL,
			state TEXT NOT NULL,
			PRIMARY KEY(character, state)
		);`,
	}
	for i, createTable := range sqlCreateTables {
		tableName := sqlTables[i]
		stmt, err := db.Prepare(createTable)
		if err != nil {
			log.Fatalf("Could not create table %s: %q.\n", tableName, err.Error())
		}
		stmt.Exec()
	}
	return
}

var stdLanguages = []dataset.Lang{
	{Code: "NS", Label: "Scientific"},
	{Code: "NV", Label: "Vernacular"},
	{Code: "CN", Label: "Chinese"},
	{Code: "EN", Label: "English"},
	{Code: "FR", Label: "French"},
}

func InsertStandardContent(db *sql.DB) error {
	if insertLang, err := db.Prepare(`INSERT INTO Languages (code, label) VALUES (?,?)`); err == nil {
		for _, lang := range stdLanguages {
			_, err = insertLang.Exec(lang.Code, lang.Label)
			if err != nil {
				return err
			}
		}
	} else {
		return err
	}
	return nil
}

type DatasetRegistry struct {
	db              *sql.DB
	picCount        int
	taxonsCount     int
	charactersCount int
}

func NewRegistry(db *sql.DB) *DatasetRegistry {
	return &DatasetRegistry{db: db}
}

const (
	QUERY_INSERT_ITEM        = `INSERT INTO Items (id, ord, name, description) VALUES (?,?,?,?);`
	QUERY_INSERT_NAMES       = `INSERT INTO ItemNames (item, lang, text) VALUES (?,?,?);`
	QUERY_INSERT_HIERARCHIES = `INSERT INTO Hierarchies (ancestor, descendant, length)
		SELECT ancestor, ?, length + 1 FROM Hierarchies
		WHERE descendant = ?
		UNION ALL
		SELECT ?, ?, 0;`
)

func (reg *DatasetRegistry) insertHierarchicalItem(op *DatabaseOperation, insertItem *sql.Stmt, insertNames *sql.Stmt, insertHierarchy *sql.Stmt,
	order int, hierarchy *dataset.Hierarchy, parent *dataset.Hierarchy) error {
	var parentId string
	if parent != nil {
		parentId = parent.Id
	}
	op.TryExec(insertItem, hierarchy.Id, order, hierarchy.Name.Scientific, hierarchy.Description)
	for lang, text := range hierarchy.Name.NamesByLangRef {
		op.TryExec(insertNames, hierarchy.Id, lang, text)
	}
	op.TryExec(insertHierarchy, hierarchy.Id, parentId, hierarchy.Id, hierarchy.Id)
	return op.Error()
}

type insertCharacterPreparedStatements struct {
	insertItem           *sql.Stmt
	insertItemNames      *sql.Stmt
	insertItemPicture    *sql.Stmt
	insertHierarchy      *sql.Stmt
	insertCharacter      *sql.Stmt
	insertState          *sql.Stmt
	insertRequiredStates *sql.Stmt
}

func (reg *DatasetRegistry) recursivelyInsertCharacters(ds *dataset.Dataset, op *DatabaseOperation, stmts insertCharacterPreparedStatements, character *dataset.Character, parentHierarchy *dataset.Hierarchy) error {
	reg.insertHierarchicalItem(op, stmts.insertItem, stmts.insertItemNames, stmts.insertHierarchy, reg.charactersCount, character.Hierarchy, parentHierarchy)
	reg.charactersCount++
	op.TryExec(stmts.insertCharacter, character.Id)
	for _, pic := range character.Pictures {
		reg.picCount++
		op.TryExec(stmts.insertItemPicture, reg.picCount, character.Id, pic.Source, pic.Legend)
	}
	for i, state := range character.States {
		op.TryExec(stmts.insertItem, state.Id, i, state.Name.Scientific, state.Description)
		for lang, text := range state.Name.NamesByLangRef {
			op.TryExec(stmts.insertItemNames, state.Id, lang, text)
		}
		op.TryExec(stmts.insertState, state.Id, character.Id)
		for _, pic := range state.Pictures {
			reg.picCount++
			op.TryExec(stmts.insertItemPicture, reg.picCount, state.Id, pic.Source, pic.Legend)
		}
	}
	for _, child := range character.Children {
		ch, ok := ds.CharactersById[child.Id]
		if !ok {
			ch = &dataset.Character{Hierarchy: child}
		}
		reg.recursivelyInsertCharacters(ds, op, stmts, ch, character.Hierarchy)
	}
	for _, state := range character.RequiredStates {
		op.TryExec(stmts.insertRequiredStates, character.Id, state.Id)
	}
	return op.Error()
}

func (reg *DatasetRegistry) insertCharacters(ds *dataset.Dataset, character *dataset.Character, parent *dataset.Character) error {
	var parentHierarchy *dataset.Hierarchy
	if parent != nil {
		parentHierarchy = parent.Hierarchy
	}
	op := NewDatabaseOperation(reg.db)
	defer op.Close()
	stmts := insertCharacterPreparedStatements{
		insertItem:           op.TryPrepare(QUERY_INSERT_ITEM),
		insertItemNames:      op.TryPrepare(QUERY_INSERT_NAMES),
		insertItemPicture:    op.TryPrepare(`INSERT INTO ItemPictures (id,item,url,label) VALUES (?,?,?,?);`),
		insertHierarchy:      op.TryPrepare(QUERY_INSERT_HIERARCHIES),
		insertCharacter:      op.TryPrepare(`INSERT INTO Characters (item) VALUES (?);`),
		insertState:          op.TryPrepare(`INSERT INTO STATES (item,character) VALUES (?,?);`),
		insertRequiredStates: op.TryPrepare(`INSERT INTO CharacterRequiredStates (character,state) VALUES (?,?);`),
	}
	return reg.recursivelyInsertCharacters(ds, op, stmts, character, parentHierarchy)
}

type insertTaxonPreparedStatements struct {
	insertItem        *sql.Stmt
	insertItemNames   *sql.Stmt
	insertItemPicture *sql.Stmt
	insertHierarchy   *sql.Stmt
	insertTaxon       *sql.Stmt
	insertTaxonStates *sql.Stmt
}

func (reg *DatasetRegistry) recursivelyInsertTaxons(ds *dataset.Dataset, op *DatabaseOperation, stmts insertTaxonPreparedStatements, taxon *dataset.Taxon, parentHierarchy *dataset.Hierarchy) error {
	reg.insertHierarchicalItem(op, stmts.insertItem, stmts.insertItemNames, stmts.insertHierarchy, reg.taxonsCount, taxon.Hierarchy, parentHierarchy)
	reg.taxonsCount++
	op.TryExec(stmts.insertTaxon, taxon.Id, taxon.Author)
	for _, pic := range taxon.Pictures {
		reg.picCount++
		op.TryExec(stmts.insertItemPicture, reg.picCount, taxon.Id, pic.Source, pic.Legend)
	}
	for _, state := range taxon.States {
		op.TryExec(stmts.insertTaxonStates, taxon.Id, state.Id)
	}
	for _, child := range taxon.Children {
		t, ok := ds.TaxonsById[child.Id]
		if !ok {
			t = dataset.NewTaxon(child)
		}
		reg.recursivelyInsertTaxons(ds, op, stmts, t, taxon.Hierarchy)
	}
	return op.Error()
}

func (reg *DatasetRegistry) insertTaxons(ds *dataset.Dataset, taxon *dataset.Taxon, parent *dataset.Taxon) error {
	var parentHierarchy *dataset.Hierarchy
	if parent != nil {
		parentHierarchy = parent.Hierarchy
	}
	op := NewDatabaseOperation(reg.db)
	defer op.Close()
	stmts := insertTaxonPreparedStatements{
		insertItem:        op.TryPrepare(QUERY_INSERT_ITEM),
		insertItemNames:   op.TryPrepare(QUERY_INSERT_NAMES),
		insertItemPicture: op.TryPrepare(`INSERT INTO ItemPictures (id,item,url,label) VALUES (?,?,?,?);`),
		insertHierarchy:   op.TryPrepare(QUERY_INSERT_HIERARCHIES),
		insertTaxon:       op.TryPrepare(`INSERT INTO Taxons (item, author) VALUES (?,?);`),
		insertTaxonStates: op.TryPrepare(`INSERT INTO TaxonStates (taxon, state) VALUES (?,?);`),
	}
	return reg.recursivelyInsertTaxons(ds, op, stmts, taxon, parentHierarchy)
}

func (reg *DatasetRegistry) InsertDataset(ds *dataset.Dataset) (err error) {
	if err = reg.insertCharacters(ds, dataset.NewCharacter(ds.CharactersHierarchy), nil); err != nil {
		log.Fatalf("Cannot insert hierarchy: %q.\n", err.Error())
	}
	if err = reg.insertTaxons(ds, dataset.NewTaxon(ds.TaxonsHierarchy), nil); err != nil {
		log.Fatalf("Cannot insert hierarchy: %q.\n", err.Error())
	}
	return err
}

func inLen(length int) string {
	var b strings.Builder
	var sep string
	for i := 0; i < length; i++ {
		b.WriteString(sep)
		sep = ","
		b.WriteString("?")
	}
	return b.String()
}

func strSliceToInterface(strs []string) []interface{} {
	anys := make([]interface{}, len(strs))
	for i, str := range strs {
		anys[i] = str
	}
	return anys
}

func (reg *DatasetRegistry) GetTaxonsHavingStates(states []string) ([]*dataset.Taxon, error) {
	op := NewDatabaseOperation(reg.db)
	defer op.Close()
	selectTaxons := op.TryPrepare(fmt.Sprintf(
		`SELECT Taxon.id, Taxon.name
		FROM Items Taxon
		INNER JOIN TaxonStates ON Taxon.id = TaxonStates.taxon 
		WHERE TaxonStates.state IN (%s)
		GROUP BY Taxon.id
		HAVING Count(TaxonStates.state) = ?`, inLen(len(states))))
	args := strSliceToInterface(states)
	args = append(args, len(states))
	rows := op.TryQuery(selectTaxons, args...)
	if op.HasFailed() {
		return nil, op.Error()
	}
	defer rows.Close()
	taxons := []*dataset.Taxon{}
	for rows.Next() {
		taxon := dataset.NewTaxon(&dataset.Hierarchy{})
		rows.Scan(&taxon.Id, &taxon.Name.Scientific)
		taxons = append(taxons, taxon)
	}
	return taxons, nil
}

func (reg *DatasetRegistry) GetAllCharactersExcept(characterIds []string) ([]*dataset.Character, map[string]*dataset.Character, error) {
	op := NewDatabaseOperation(reg.db)
	defer op.Close()
	selectCharacters := op.TryPrepare(fmt.Sprintf(
		`SELECT Character.id, Character.name, CharName.lang, CharName.text, CharPic.id, CharPic.url, 
		State.id, State.name, StateName.lang, StateName.text, Hierarchies.ancestor, StatePic.id, StatePic.url
		FROM Items Character
		INNER JOIN Characters ON Characters.item = Character.id
		LEFT JOIN ItemNames CharName ON CharName.item = Character.id
		LEFT JOIN ItemPictures CharPic ON CharPic.item = Character.id
		INNER JOIN States ON States.character = Character.id
		INNER JOIN Items State ON State.id = States.item
		LEFT JOIN ItemNames StateName ON StateName.item = State.id
		LEFT JOIN ItemPictures StatePic ON StatePic.item = State.id
		LEFT JOIN Hierarchies ON Hierarchies.descendant = Character.id
		WHERE Hierarchies.length = 1 AND NOT Character.id IN (%s)
		ORDER BY Character.ord ASC, State.ord ASC`, inLen(len(characterIds))))
	rows := op.TryQuery(selectCharacters, strSliceToInterface(characterIds)...)
	if op.HasFailed() {
		return nil, nil, op.Error()
	}
	defer rows.Close()
	characters := []*dataset.Character{}
	charactersById := map[string]*dataset.Character{}
	charIdsByParentIds := map[string][]string{}
	var lastCharacter *dataset.Character
	var lastState *dataset.State
	var lastCharPic int64
	var lastCharLang string
	var lastStatePic string
	var lastStateLang string

	for rows.Next() {
		var charPicId sql.NullInt64
		var charLang, charLangName, charPicUrl, stateLang, stateLangName, statePicId, statePicUrl sql.NullString
		var charId, charName, stateId, stateName, parentId string
		err := rows.Scan(&charId, &charName, &charLang, &charLangName, &charPicId, &charPicUrl,
			&stateId, &stateName, &stateLang, &stateLangName, &parentId, &statePicId, &statePicUrl)
		if err != nil {
			fmt.Println(err.Error())
		}
		if lastCharacter == nil || lastCharacter.Id != charId {
			charIdsByParentIds[parentId] = append(charIdsByParentIds[parentId], charId)
			lastCharacter = dataset.NewCharacter(&dataset.Hierarchy{
				Id: charId,
				Name: dataset.MultilangText{
					Scientific:     charName,
					NamesByLangRef: make(map[string]string),
				},
			})
			charactersById[charId] = lastCharacter
			if parentId == "c0" {
				characters = append(characters, lastCharacter)
			}
		}
		if charLang.Valid && charLang.String != lastCharLang {
			lastCharLang = charLang.String
			lastCharacter.Name.NamesByLangRef[lastCharLang] = charLangName.String
		}
		if charPicId.Valid && (lastCharPic == 0 || lastCharPic != charPicId.Int64) {
			lastCharPic = charPicId.Int64
			lastCharacter.Pictures = append(lastCharacter.Pictures, dataset.Picture{Id: fmt.Sprint(lastCharPic), Source: charPicUrl.String})
		}
		if lastState == nil || lastState.Id != stateId {
			lastCharacter.States = append(lastCharacter.States, dataset.State{
				Id: stateId,
				Name: dataset.MultilangText{
					Scientific:     stateName,
					NamesByLangRef: make(map[string]string),
				},
			})
			lastState = &lastCharacter.States[len(lastCharacter.States)-1]
		}
		if stateLang.Valid {
			lastStateLang = stateLang.String
			lastState.Name.NamesByLangRef[lastStateLang] = stateLangName.String
		}
		if statePicId.Valid && (len(lastStatePic) == 0 || lastStatePic != statePicId.String) {
			lastStatePic = statePicId.String
			lastState.Pictures = append(lastState.Pictures, dataset.Picture{Id: lastStatePic, Source: statePicUrl.String})
		}
	}
	for parentId, childrenIds := range charIdsByParentIds {
		if parent, ok := charactersById[parentId]; ok {
			for _, childId := range childrenIds {
				if child, ok := charactersById[childId]; ok {
					parent.Children = append(parent.Children, child.Hierarchy)
				}
			}
		}
	}
	return characters, charactersById, nil
}

func (reg *DatasetRegistry) GetCharactersFromIds(ids []string, stateIds []string) ([]*dataset.Character, error) {
	characters := make([]*dataset.Character, 0, len(ids))
	op := NewDatabaseOperation(reg.db)
	selectCharacters := op.TryPrepare(
		fmt.Sprintf(`SELECT Character.id, Character.name, State.id, State.name FROM Items Character 
				LEFT JOIN States ON States.character = Character.id
				LEFT JOIN Items State ON State.id = States.item
				WHERE Character.id IN (%s) AND State.id IN (%s)
				ORDER BY Character.id`,
			inLen(len(ids)), inLen(len(stateIds))))
	rows := op.TryQuery(selectCharacters, append(strSliceToInterface(ids), strSliceToInterface(stateIds)...)...)
	if op.HasFailed() {
		return nil, op.Error()
	}
	defer rows.Close()
	var lastChar *dataset.Character
	var lastState *dataset.State
	for rows.Next() {
		var Charid, charName, stateId, stateName string
		rows.Scan(&Charid, &charName, &stateId, &stateName)
		if lastChar == nil || lastChar.Id != Charid {
			lastChar = dataset.NewCharacter(&dataset.Hierarchy{Id: Charid, Name: dataset.MultilangText{Scientific: charName}})
			characters = append(characters, lastChar)
		}
		if lastState == nil || lastState.Id != stateId {
			lastChar.States = append(lastChar.States, dataset.State{Id: stateId, Name: dataset.MultilangText{Scientific: stateName}})
			lastState = &lastChar.States[len(lastChar.States)-1]
		}
	}
	return characters, nil
}

type picture struct {
	url     string
	content []byte
	err     error
}

func requestImage(url string, ch chan<- picture) {
	resp, _ := http.Get(url)
	resp, err := http.Get(url)
	if err != nil {
		ch <- picture{url: url, content: []byte{}, err: err}
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		ch <- picture{url: url, content: []byte{}, err: err}
		return
	}
	ch <- picture{url: url, content: body, err: nil}
}

func (reg *DatasetRegistry) CacheImages() error {
	op := NewDatabaseOperation(reg.db)
	defer op.Close()
	selectPics := op.TryPrepare(`SELECT url FROM ItemPictures;`)
	insertCache := op.TryPrepare(`INSERT OR REPLACE INTO PictureCache (src, data) VALUES (?,?)`)
	rows := op.TryQuery(selectPics)
	ch := make(chan picture)
	count := 0
	for rows.Next() {
		count++
		var url string
		rows.Scan(&url)
		go requestImage(url, ch)
	}
	for i := 0; i < count; i++ {
		pic := <-ch
		if pic.err != nil {
			continue
		}
		op.TryExec(insertCache, pic.url, pic.content)
	}
	return op.Error()
}

func (reg *DatasetRegistry) GetCachedImage(url string) ([]byte, bool) {
	op := NewDatabaseOperation(reg.db)
	defer op.Close()
	selectCache := op.TryPrepare(`SELECT data FROM PictureCache WHERE src = ?`)
	rows := op.TryQuery(selectCache, url)
	if op.Error() != nil {
		fmt.Printf("Error retrieving cached image: %q", op.Error().Error())
		return nil, false
	}
	for rows.Next() {
		var data []byte
		err := rows.Scan(&data)
		if err != nil {
			fmt.Printf("Error retrieving cached image: %q", err.Error())
			return nil, false
		}
		return data, true
	}
	fmt.Printf("No such image: %q", url)
	return nil, false
}
