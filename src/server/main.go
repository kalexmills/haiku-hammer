package main

import (
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer"
	"github.com/kalexmills/haiku-enforcer/src/haikuhammer/db"
	"github.com/spf13/viper"
	"log"

	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf := readConfig()
	hh := haikuhammer.NewHaikuHammer(conf)

	err := hh.Open()
	if err != nil {
		log.Fatalf("fail error opening bot: %v", err)
	}

	log.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	err = hh.Close()
	if err != nil {
		log.Println("error closing session,", err)
	}
}

func readConfig() haikuhammer.Config {
	viper.SetDefault("reactHaiku", true)
	viper.SetDefault("reactNonHaiku", false)
	viper.SetDefault("deleteNonHaiku", false)
	viper.SetDefault("explainNonHaiku", true)
	viper.SetDefault("serveRandomHaiku", true)
	viper.SetDefault("positiveReacts", []string{"💯","🍙","🍵","🍶","🍜"})
	viper.SetDefault("negativeReacts", []string{"🚫","⛔"})
	viper.SetDefault("dbPath", "./haikuDB.sqlite3")
	viper.SetDefault("debug", false)

	viper.SetEnvPrefix("HAIKU_HAMMER")
	viper.AutomaticEnv()

	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/haikuhammer")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("no config file found, using defaults,", err)
	}
	flags := db.ConfigFlag(0)
	if viper.GetBool("reactHaiku") {
		flags |= db.ConfigReactToHaiku
	}
	if viper.GetBool("reactNonHaiku") {
		flags |= db.ConfigReactToNonHaiku
	}
	if viper.GetBool("deleteNonHaiku") {
		flags |= db.ConfigDeleteNonHaiku
	}
	if viper.GetBool("explainNonHaiku") {
		flags |= db.ConfigExplainNonHaiku
	}
	if viper.GetBool("serveRandomHaiku") {
		flags |= db.ConfigServeRandomHaiku
	}
	return haikuhammer.Config{
		Token: viper.GetString("token"),
		ActionFlags: flags,
		PositiveReacts: viper.GetStringSlice("positiveReacts"),
		NegativeReacts: viper.GetStringSlice("negativeReacts"),
		Debug: viper.GetBool("debug"),
		DBPath: viper.GetString("dbPath"),
	}
}