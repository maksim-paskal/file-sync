package main

import "flag"

type Config struct {
	httpAddress    *string
	httpsAddress   *string
	sourceDir      *string
	destinationDir *string
}

func newConfig() *Config {
	config := Config{
		httpAddress:    flag.String("http.address", ":9336", "address"),
		httpsAddress:   flag.String("https.address", ":9335", "address"),
		sourceDir:      flag.String("dir.src", "data", "folder"),
		destinationDir: flag.String("dir.dest", "data", "folder"),
	}

	flag.Parse()

	return &config
}
