package main

import (
	"flag"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"simulator"
	"time"
)

type FlagStringSlice []string

func (f *FlagStringSlice) String() string {
	return "[]string"
}

func (f *FlagStringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}

// TODO: find a kafka producer
// https://github.com/ORBAT/krater/ looks promising

// use github.com/hamba/avro
// Encoder to setup avro writing for each of the topics

func main() {
	rand.Seed(time.Now().UnixNano())

	configPaths := FlagStringSlice{}
	cpuprofile := ""

	flag.Var(&configPaths, "config", "path to the config file; can be provided multiple times, files will be merged in the order provided")
	flag.StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to `file`")

	flag.Parse()

	if len(configPaths) == 0 {
		configPaths.Set("config.yaml")
	}

	log.SetFlags(log.Ldate | log.Ltime)

	config, err := simulator.ParseConfigs([]string(configPaths))
	if err != nil {
		log.Fatalf("unable to load config files: %v; error: %+v", configPaths, err)
	}

	if cpuprofile != "" {
		// disable logging and lower verbosity during profile
		log.SetOutput(ioutil.Discard)
		config.Verbose = 0

		f, err := os.Create(cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	// TODO: serve grafana metrics

	var db simulator.Database
	for {
		db, err = simulator.NewSingleStore(config.Database)
		if err != nil {
			log.Printf("unable to connect to SingleStore: %s; retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		break
	}

	locations, err := db.Locations()
	if err != nil {
		log.Fatalf("unable to download locations from SingleStore: %+v", err)
	}
	index, err := simulator.NewLocationIndexFromDB(locations)
	if err != nil {
		log.Fatalf("unable to build location index: %+v", err)
	}

	packages, err := db.ActivePackages()
	if err != nil {
		log.Fatalf("unable to download packages from SingleStore: %+v", err)
	}

	trackers, err := simulator.NewTrackersFromActivePackages(config, index, packages)
	if err != nil {
		log.Fatalf("unable to download locations from SingleStore: %+v", err)
	}

	state := simulator.NewState(config, index, trackers)

	simulator.Simulate(state)
}
