//go:generate giddyup --variable=version
package main

import (
	"github.com/alexandre-normand/slackscot"
	"github.com/alexandre-normand/slackscot/config"
	"github.com/alexandre-normand/slackscot/plugins"
	"github.com/alexandre-normand/slackscot/store"
	"github.com/alexandre-normand/slackscot/store/datastoredb"
	"github.com/alexandre-normand/slackscot/store/inmemorydb"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
	"gopkg.in/alecthomas/kingpin.v2"
	"log"
	"os"
)

var (
	configurationPath  = kingpin.Flag("configuration", "The path to the configuration file.").Required().String()
	gcpCredentialsFile = kingpin.Flag("gcpCredentialsFile", "The path to the google cloud json credentials file.").String()
	logfile            = kingpin.Flag("log", "The path to the log file").OpenFile(os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
)

const (
	storagePathKey        = "storagePath" // Root directory for the file-based leveldb storage
	gcpProjectIDConfigKey = "gcpProjectID"
	name                  = "youppi"
)

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	v := config.NewViperWithDefaults()
	// Enable getting configuration from the environment, especially useful for the slack token
	v.AutomaticEnv()
	// Bind the token key config to the env variable SLACK_TOKEN (case sensitive)
	v.BindEnv(config.TokenKey, "SLACK_TOKEN")
	v.BindEnv(gcpProjectIDConfigKey, "GCP_PROJECT_ID")

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

	youppi, err := slackscot.New("youppi", v, options...)
	if err != nil {
		log.Fatal(err)
	}

	storagePath := v.GetString(storagePathKey)
	gcpProjectID := v.GetString(gcpProjectIDConfigKey)

	karmaStorer, err := newStorer(plugins.KarmaPluginName, storagePath, gcpProjectID, *gcpCredentialsFile)
	if err != nil {
		log.Fatalf("Opening opening db for [%s]: %s", plugins.KarmaPluginName, err.Error())
	}
	defer karmaStorer.Close()

	karma := plugins.NewKarma(karmaStorer)
	youppi.RegisterPlugin(&karma.Plugin)

	triggererStorer, err := newStorer("triggerer", storagePath, gcpProjectID, *gcpCredentialsFile)
	if err != nil {
		log.Fatalf("Opening opening db for [%s]: %s", "triggerer", err.Error())
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

	versionner := plugins.NewVersionner("youppi", version)
	youppi.RegisterPlugin(&versionner.Plugin)

	err = youppi.Run()
	if err != nil {
		log.Fatal(err)
	}
}

// newStorer returns a new StringStorer for a plugin and is responsible for deciding whether to create a leveldb
// implementation or a inmemorydb/datastoredb one. The latter is preferred but we have the fallback if gcpProjectID
// is not set so that local developers have an easy way to run it when developing
func newStorer(pluginName string, storagePath string, gcpProjectID string, gcpCredentialsFile string) (storer store.GlobalSiloStringStorer, err error) {
	if gcpProjectID != "" {
		return newDatastoreStorerWithInMemoryCache(pluginName, gcpProjectID, gcpCredentialsFile)
	}

	return store.NewLevelDB(pluginName, storagePath)
}

// newDatastoreStorerWithInMemoryCache creates a new instance of a Google Cloud Datastore DB wrapped with an in-memory db that
// acts as a write-through cache to prevent hitting the datastore. This is especially useful with plugins that use their
// StringStorer to do their matching and answering logic that can get called quite often
func newDatastoreStorerWithInMemoryCache(pluginName string, gcpProjectID string, gcpCredentialsFile string) (storer store.GlobalSiloStringStorer, err error) {
	gcloudKarmaStorer, err := datastoredb.New(pluginName, gcpProjectID, option.WithCredentialsFile(gcpCredentialsFile))
	if err != nil {
		return nil, err
	}

	storer, err = inmemorydb.New(gcloudKarmaStorer)
	if err != nil {
		return nil, err
	}

	return storer, err
}
