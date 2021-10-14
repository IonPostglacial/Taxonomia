package cmd

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"nicolas.galipot.net/taxonomia/dataset"
	"nicolas.galipot.net/taxonomia/dataset/database"
	"nicolas.galipot.net/taxonomia/dataset/identification"
)

func getDatabaseOrDie(dbPath string) *sql.DB {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Cannot open database: %q.\n", err.Error())
	}
	return db
}

func Initialize() {
	db := getDatabaseOrDie("db.sq3")
	defer db.Close()
	if err := database.CreateTables(db); err != nil {
		log.Fatalf("Cannot create tables: %q.\n", err.Error())
	}
	if err := database.InsertStandardContent(db); err != nil {
		log.Fatalf("Cannot insert standard content: %q.\n", err.Error())
	}
}

func getDatasetFilePath() string {
	dsName := "dataset.hazo.json"
	if len(os.Args) > 2 {
		dsName = os.Args[2]
	}
	return dsName
}

func Import() {
	dsName := getDatasetFilePath()
	f, err := os.Open(dsName)
	if err == nil {
		defer f.Close()
	} else {
		log.Fatalf("Cannot read file '%s': '%s'", dsName, err.Error())
	}
	r := bufio.NewReader(f)
	ds, err := dataset.ReadHazo(r)
	if err != nil {
		log.Fatalf("Cannot read Hazo dataset file: '%s'\n", err.Error())
	}
	db := getDatabaseOrDie("db.sq3")
	defer db.Close()
	reg := database.NewRegistry(db)
	reg.InsertDataset(ds)
}

func CacheImages() {
	db := getDatabaseOrDie("db.sq3")
	reg := database.NewRegistry(db)
	if err := reg.CacheImages(); err != nil {
		log.Fatalf("Error caching images: %q", err.Error())
	}
}

func displayCharacter(charactersByids map[string]*dataset.Character, ch *dataset.Character, indentation string) {
	if ch != nil {
		fmt.Println(indentation, ch.Id, ch.Name.Scientific)
		for _, state := range ch.States {
			fmt.Println(indentation, "-", state.Id, state.Name.Scientific)
		}
		for _, item := range ch.Children {
			if child, ok := charactersByids[item.Id]; ok {
				displayCharacter(charactersByids, child, indentation+indentation)
			}
		}
	}
}

func ListCharacters() {
	db := getDatabaseOrDie("db.sq3")
	reg := database.NewRegistry(db)
	characters, err := reg.GetAllCharactersExcept([]string{})
	if err != nil {
		log.Fatalf("Cannot list characters: %q.\n", err.Error())
	}
	for _, ch := range characters {
		displayCharacter(characters, ch, " |")
	}
}

func Identify() {
	db := getDatabaseOrDie("db.sq3")
	reg := database.NewRegistry(db)
	characters, err := reg.GetAllCharactersExcept([]string{})
	if err != nil {
		log.Fatalf("Cannot retrieve characters: %q.\n", err.Error())
	}
	selectedStates := []string{}
	for _, character := range characters {
		stateIds := []string{}
		if len(characters) > 0 {
			fmt.Printf("How is %s?\n", character.Name.Scientific)
		}
		for i, state := range character.States {
			stateIds = append(stateIds, state.Id)
			fmt.Printf("%d - %s\n", i+1, state.Name.Scientific)
		}
		var index int
		nbSucc, err := fmt.Scanf("%d", &index)
		if err != nil || nbSucc < 1 {
			fmt.Println("Wrong input")
			continue
		}
		if index > 0 && index <= len(stateIds) {
			selectedStates = append(selectedStates, stateIds[index-1])
		} else {
			fmt.Println("Index out of bounds", index)
			continue
		}
		taxons, err := reg.GetTaxonsHavingStates(selectedStates)
		if err != nil {
			log.Fatalf("Cannot retrieve taxons: %q.\n", err.Error())
		}
		if len(taxons) == 0 {
			fmt.Println("there are no results")
			return
		}
		fmt.Println("results:")
		for _, taxon := range taxons {
			fmt.Println(taxon.Name.Scientific)
		}
	}
}

func Serve(args []string) {
	serveFS := flag.NewFlagSet("server", flag.ExitOnError)
	key := serveFS.String("key", "", "Cookie store session key")
	dbPath := serveFS.String("db", "db.sq3", "Path to to database file")
	hostname := serveFS.String("host", "localhost", "The name of the host serving the app.")
	port := serveFS.String("port", "8080", "The port where the app is served.")
	serveFS.Parse(args)
	fmt.Printf("serving hostname: %s, port: %s\n", *hostname, *port)
	db := getDatabaseOrDie(*dbPath)
	defer db.Close()
	reg := database.NewRegistry(db)
	identificationHandler := identification.NewHandler(reg, *key)
	http.HandleFunc("/static/", dataset.StaticHandler)
	http.HandleFunc("/img", database.CachedImageHandler(reg))
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {})
	http.HandleFunc("/identify", identificationHandler.Func)
	http.ListenAndServe(*hostname+":"+*port, nil)
}

func Check() {
	dsName := getDatasetFilePath()
	f, err := os.Open(dsName)
	if err == nil {
		defer f.Close()
	} else {
		log.Fatalf("Cannot read file '%s': '%s'", dsName, err.Error())
	}
	r := bufio.NewReader(f)
	ds, err := dataset.ReadHazo(r)
	if err != nil {
		log.Fatalf("Cannot read Hazo dataset file: '%s'\n", err.Error())
	}
	itemIds := make(map[string]bool, len(ds.TaxonsById)+len(ds.CharactersById))
	for id := range ds.TaxonsById {
		if _, ok := itemIds[id]; ok {
			fmt.Println("duplicate key", id)
		}
		itemIds[id] = true
	}
	for id, ch := range ds.CharactersById {
		if _, ok := itemIds[id]; ok {
			fmt.Println("duplicate key", id)
		}
		for _, state := range ch.States {
			if _, ok := itemIds[state.Id]; ok {
				fmt.Println("duplicate key", state.Id)
			}
		}
		itemIds[id] = true
	}
	for id := range itemIds {
		fmt.Println(id)
	}
}
