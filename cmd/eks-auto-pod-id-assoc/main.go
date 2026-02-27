// Package main implements the tool.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/udhos/boilerplate/boilerplate"
	"github.com/udhos/boilerplate/envconfig"
)

func main() {

	//
	// command-line
	//
	var showVersion bool
	flag.BoolVar(&showVersion, "version", showVersion, "show version")
	flag.Parse()

	me := filepath.Base(os.Args[0])
	infof("%s", boilerplate.LongVersion(me))
	env := envconfig.NewSimple(me)

	//
	// version
	//
	{
		v := boilerplate.LongVersion(me + " version=" + version)
		if showVersion {
			fmt.Print(v)
			fmt.Println()
			return
		}
		infof("%s", v)
	}

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
