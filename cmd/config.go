package main

import "flag"

type Config struct {
	logLevel       *string
	httpAddress    *string
	httpsAddress   *string
	sourceDir      *string
	destinationDir *string
	syncAddress    *string
}

func newConfig() *Config {
	config := Config{
		logLevel:       flag.String("log.level", "INFO", "logging level"),
		httpAddress:    flag.String("http.address", ":9336", "address"),
		httpsAddress:   flag.String("https.address", ":9335", "address"),
		sourceDir:      flag.String("dir.src", "data", "folder"),
		destinationDir: flag.String("dir.dest", "data", "folder"),
		syncAddress:    flag.String("sync.address", "localhost:9335", "destination server"),
	}

	flag.Parse()

	return &config
}
