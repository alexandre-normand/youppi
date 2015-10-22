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

	var youppi Youppi

	// Registers deferred call to close resources on shutdown
	defer youppi.Close()

	slackscot.Run(youppi, *config)
}

type Youppi struct {
	bundles []slackscot.ExtentionBundle
}

func (youppi Youppi) Init(config config.Configuration) (commands []slackscot.Action, listeners []slackscot.Action, err error) {
	karma := brain.NewKarma()
	c, l, err := karma.Init(config)
	if err != nil {
		return nil, nil, err
	}

	commands = append(commands, c...)
	listeners = append(listeners, l...)

	imager := brain.NewImager()
	c, l, err = imager.Init(config)
	if err != nil {
		return nil, nil, err
	}
	commands = append(commands, c...)
	listeners = append(listeners, l...)

	//initImagesExt(config)
	//initServiceCheck(config)
	return commands, listeners, nil

}

func (youppi Youppi) Close() {
	// Close any resources needed by scripts
	for _, b := range youppi.bundles {
		b.Close()
	}
}
