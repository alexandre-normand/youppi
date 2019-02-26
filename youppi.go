//go:generate giddyup --variable=version
package main

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/spf13/viper"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
)

var (
	configurationPath = kingpin.Flag("configuration", "The path to the configuration file.").Required().String()
	logfile           = kingpin.Flag("log", "The path to the log file").OpenFile(os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
)

const (
	storagePathKey = "storagePath" // Root directory for the file-based leveldb storage
	name           = "youppi"
)

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	v := config.NewViperWithDefaults()
	// Enable getting configuration from the environment, especially useful for the slack token
	v.AutomaticEnv()
	// Bind the token key config to the env variable SLACK_TOKEN (case sensitive)
	v.BindEnv(config.TokenKey, "SLACK_TOKEN")

	v.SetConfigFile(*configurationPath)
	err := v.ReadInConfig()
	if err != nil {
		log.Fatalf("Error loading configuration file [%s]: %v", *configurationPath, err)
	}

	// Do this only so that we can get a global debug flag available to everything
	viper.Set(config.DebugKey, v.GetBool(config.DebugKey))

	options := make([]slackscot.Option, 0)
	if *logfile != nil {
		options = append(options, slackscot.OptionLogfile(*logfile))
	}

	youppi, err := slackscot.NewSlackscot("youppi", v, options...)
	if err != nil {
		log.Fatal(err)
	}

	if !v.IsSet(storagePathKey) {
		log.Fatalf("Missing [%s] configuration key in the top value configuration", storagePathKey)
	}

	storagePath := v.GetString(storagePathKey)
	karmaStorer, err := store.NewLevelDB(plugins.KarmaPluginName, storagePath)
	if err != nil {
		log.Fatalf("Opening [%s] db failed with path [%s]", plugins.KarmaPluginName, storagePath)
	}
	defer karmaStorer.Close()

	karma := plugins.NewKarma(karmaStorer)
	youppi.RegisterPlugin(&karma.Plugin)

	triggererStorer, err := store.NewLevelDB("triggerer", storagePath)
	if err != nil {
		log.Fatalf("Opening [%s] db failed with path [%s]", "triggerer", storagePath)
	}
	defer triggererStorer.Close()

	triggerer := plugins.NewTriggerer(triggererStorer)
	youppi.RegisterPlugin(&triggerer.Plugin)

	fingerQuoterConf, err := config.GetPluginConfig(v, plugins.FingerQuoterPluginName)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	fingerQuoter, err := plugins.NewFingerQuoter(fingerQuoterConf)
	if err != nil {
		log.Fatalf("Error initializing finger quoter plugin: %v", err)
	}
	youppi.RegisterPlugin(&fingerQuoter.Plugin)

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

	versioner := plugins.NewVersioner("youppi", version)
	youppi.RegisterPlugin(&versioner.Plugin)

	err = youppi.Run()
	if err != nil {
		log.Fatal(err)
	}
}
