package main

import (
	"os"

	_ "github.com/mattn/go-sqlite3"
	"nicolas.galipot.net/taxonomia/dataset/cmd"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cmd.Initialize()
		case "import":
			cmd.Import()
		case "cache":
			cmd.CacheImages()
		case "identify":
			cmd.Identify()
		case "lschar":
			cmd.ListCharacters()
		case "serve":
			cmd.Serve(os.Args[1:])
		}
	}
}
