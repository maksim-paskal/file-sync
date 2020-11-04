package main

import "flag"

type Config struct {
	RedisAddress          *string
	DefaultFilePermission *string
	DefaultDirPermission  *string
}

func newConfig() *Config {
	config := Config{
		RedisAddress:          flag.String("redis.address", "localhost:6379", "address"),
		DefaultFilePermission: flag.String("file.permission", "0644", "permission"),
		DefaultDirPermission:  flag.String("dir.permission", "511", "permission"),
	}

	flag.Parse()

	return &config
}
