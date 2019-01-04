//go:generate giddyup
package main

import (
	"github.com/alecthomas/kingpin"
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/spf13/viper"
	"log"
)

var (
	configurationPath = kingpin.Flag("configuration", "The path to the configuration file.").Required().String()
)

func main() {
	kingpin.Version(VERSION)
	kingpin.Parse()

	v := config.NewViperWithDefaults()
	v.SetConfigFile(*configurationPath)
	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Error loading configuration file [%s]: %v", *configurationPath, err)
	}

	// Do this only so that we can get a global debug flag available to everything
	// TODO: clean this up
	viper.Set(config.DebugKey, v.GetBool(config.DebugKey))

	youppi, err := slackscot.NewSlackscot("youppi", v)
	if err != nil {
		log.Fatal(err)
	}

	karma, err := plugins.NewKarma(v)
	if err != nil {
		log.Fatalf("Error initializing karma plugin: %v", err)
	}
	defer karma.Close()
	youppi.RegisterPlugin(&karma.Plugin)

	fingerQuoterConf, err := config.GetPluginConfig(v, plugins.FingerQuoterPluginName)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	fingerQuoter, err := plugins.NewFingerQuoter(fingerQuoterConf)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	youppi.RegisterPlugin(&fingerQuoter.Plugin)

	imager := plugins.NewImager()
	youppi.RegisterPlugin(&imager.Plugin)

	emojiBannerConf, err := config.GetPluginConfig(v, plugins.EmojiBannerPluginName)
	if err != nil {
		log.Fatalf("Error initializing emoji banner plugin: %v", err)
	}
	emojiBanner, err := plugins.NewEmojiBannerMaker(emojiBannerConf)
	if err != nil {
		log.Fatalf("Error initializing emoji banner plugin: %v", err)
	}
	defer emojiBanner.Close()
	youppi.RegisterPlugin(&emojiBanner.Plugin)

	ohMondayConf, err := config.GetPluginConfig(v, plugins.OhMondayPluginName)
	if err != nil {
		log.Fatalf("Error initializing oh monday plugin: %v", err)
	}
	ohMonday, err := plugins.NewOhMonday(ohMondayConf)
	if err != nil {
		log.Fatalf("Error initializing oh monday plugin: %v", err)
	}
	youppi.RegisterPlugin(&ohMonday.Plugin)

	versioner := plugins.NewVersioner("youppi", VERSION)
	youppi.RegisterPlugin(&versioner.Plugin)

	err = youppi.Run()
	if err != nil {
		log.Fatal(err)
	}
}
