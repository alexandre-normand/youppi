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
	"io"
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

	storagePath := v.GetString(storagePathKey)
	gcpProjectID := v.GetString(gcpProjectIDConfigKey)

	karmaStorer, err := newStorer(plugins.KarmaPluginName, storagePath, gcpProjectID, *gcpCredentialsFile)
	if err != nil {
		log.Fatalf("Opening opening db for [%s]: %s", plugins.KarmaPluginName, err.Error())
	}
	defer karmaStorer.Close()

	triggererStorer, err := newStorer(plugins.TriggererPluginName, storagePath, gcpProjectID, *gcpCredentialsFile)
	if err != nil {
		log.Fatalf("Opening opening db for [%s]: %s", plugins.TriggererPluginName, err.Error())
	}
	defer triggererStorer.Close()

	youppi, err := slackscot.NewBot(name, v, options...).
		WithPlugin(plugins.NewKarma(karmaStorer)).
		WithPlugin(plugins.NewTriggerer(triggererStorer)).
		WithConfigurablePluginErr(plugins.FingerQuoterPluginName, func(conf *config.PluginConfig) (p *slackscot.Plugin, err error) { return plugins.NewFingerQuoter(conf) }).
		WithConfigurablePluginCloserErr(plugins.EmojiBannerPluginName, func(conf *config.PluginConfig) (c io.Closer, p *slackscot.Plugin, err error) {
			return plugins.NewEmojiBannerMaker(conf)
		}).
		WithConfigurablePluginErr(plugins.OhMondayPluginName, func(conf *config.PluginConfig) (p *slackscot.Plugin, err error) { return plugins.NewOhMonday(conf) }).
		WithPlugin(plugins.NewVersionner(name, version)).
		Build()
	defer youppi.Close()

	if err != nil {
		log.Fatal(err)
	}

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
