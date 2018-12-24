package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugins"
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

	youppi := slackscot.NewSlackscot("youppi", []slackscot.Plugin{plugins.NewKarma(), plugins.NewImager(), plugins.NewFingerQuoter(), plugins.NewEmojiBannerMaker()})

	err = youppi.Run(*config)
	if err != nil {
		log.Fatal(err)
	}
}
