package main

import (
	"log"
	"os"

	"github.com/liampulles/film-fest-itinerary/cmd/filmscrapecli/commands"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: %s <command> [args]", os.Args[0])
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "jff-webtickets":
		if err := commands.RunJFFWebtickets(args); err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command: %s", command)
	}
}
