package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/brain"
	"github.com/alexandre-normand/slackscot/config"
	"log"
)

var (
	configurationPath = kingpin.Flag("configuration", "The path to the configuration file.").Required().String()
)

func main() {
	kingpin.Parse()

	config, err := config.LoadConfiguration(*configurationPath)
	if err != nil {
		log.Fatal(err)
	}

	youppi := slackscot.NewSlackscot([]slackscot.ExtentionBundle{brain.NewKarma(), brain.NewImager()})

	slackscot.Run(*youppi, *config)
}
