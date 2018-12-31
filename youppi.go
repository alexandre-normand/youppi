//go:generate giddyup
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

	c := *config

	youppi, err := slackscot.NewSlackscot("youppi", c)
	if err != nil {
		log.Fatal(err)
	}

	karma, err := plugins.NewKarma(c)
	if err != nil {
		log.Fatalf("Error initializing karma plugin: %v", err)
	}
	defer karma.Close()
	youppi.RegisterPlugin(&karma.Plugin)

	fingerQuoter, err := plugins.NewFingerQuoter(c)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	youppi.RegisterPlugin(&fingerQuoter.Plugin)

	imager := plugins.NewImager(c)
	youppi.RegisterPlugin(&imager.Plugin)

	emojiBanner, err := plugins.NewEmojiBannerMaker(c)
	if err != nil {
		log.Fatalf("Error initializing emoji banner plugin: %v", err)
	}
	youppi.RegisterPlugin(&emojiBanner.Plugin)

	err = youppi.Run()
	if err != nil {
		log.Fatal(err)
	}
}
