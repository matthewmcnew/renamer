package main

import (
	"flag"
	"fmt"
	"github.com/matthewmcnew/renamer"
	"log"
)

var (
	image = flag.String("image", "", "The exisiting buildpack image")
	tag   = flag.String("tag", "", "The tag of the new buildpack image")
	id    = flag.String("id", "", "The new id of the image")
)

func main() {
	flag.Parse()

	rename, err := renamer.Rename(*image, *id, *tag)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(rename)

}
