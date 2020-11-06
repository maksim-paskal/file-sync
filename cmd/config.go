package main

import "flag"

type Config struct {
	logLevel       *string
	httpAddress    *string
	httpsAddress   *string
	sourceDir      *string
	destinationDir *string
	syncAddress    *string
	sslClientKey   *string
	sslClientCrt   *string
	sslServerKey   *string
	sslServerCrt   *string
	sslCA          *string
}

func newConfig() *Config {
	config := Config{
		logLevel:       flag.String("log.level", "INFO", "logging level"),
		httpAddress:    flag.String("http.address", ":9336", "address"),
		httpsAddress:   flag.String("https.address", ":9335", "address"),
		sourceDir:      flag.String("dir.src", "data", "folder"),
		destinationDir: flag.String("dir.dest", "data", "folder"),
		syncAddress:    flag.String("sync.address", "localhost:9335", "destination server"),
		// ssl config
		sslClientKey: flag.String("ssl.clientKey", "ssl/client01.key", "ssl certificate"),
		sslClientCrt: flag.String("ssl.clientCrt", "ssl/client01.crt", "ssl certificate"),
		sslServerKey: flag.String("ssl.serverKey", "ssl/server.key", "ssl certificate"),
		sslServerCrt: flag.String("ssl.serverCrt", "ssl/server.crt", "ssl certificate"),
		sslCA:        flag.String("ssl.serverCA", "ssl/ca.crt", "ssl certificate"),
	}

	flag.Parse()

	return &config
}
