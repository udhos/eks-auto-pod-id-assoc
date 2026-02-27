// Package main implements the tool.
package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/boilerplate/envconfig"
)

func main() {
	me := filepath.Base(os.Args[0])
	log.Println(boilerplate.LongVersion(me))
	env := envconfig.NewSimple(me)

	configFile := env.String("CONFIG_FILE", "config.yaml")
	cfg, err := loadConfigFromFile(configFile)
	if err != nil {
		fatalf("failed to load config: %s: %v", configFile, err)
	}

	interval := env.Duration("INTERVAL", time.Minute)
	once := env.Bool("RUN_ONCE", false)
	dry := env.Bool("DRY", true)

	app := newApplication(cfg, newRealClient(me, dry))

	for {
		app.run()

		if once {
			infof("RUN_ONCE=true, exiting")
			break
		}

		infof("sleeping %v", interval)
		time.Sleep(interval)
	}
}
