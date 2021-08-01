package main

import (
	"bytes"
	"flag"
	"log"
	"os"

	"github.com/traefik/yaegi/extract"
)

const pkgIdent = "github.com/glasslabs/looking-glass/module/types"

var outputFile = flag.String("o", "gen.go", "The output file")

func main() {
	flag.Parse()

	ext := extract.Extractor{Dest: "types"}

	var buf bytes.Buffer
	_, err := ext.Extract(pkgIdent, "", &buf)
	if err != nil {
		log.Println(err)
		return
	}

	if err = os.WriteFile(*outputFile, buf.Bytes(), 0600); err != nil {
		log.Println(err)
		return
	}
}
