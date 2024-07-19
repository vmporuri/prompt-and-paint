package main

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// Holds the configuration information for the Go server.
type Config struct {
	Server struct {
		Port string `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`
	Database struct {
		RedisHost string `yaml:"redisHost"`
		RedisPort string `yaml:"redisPort"`
	} `yaml:"database"`
}

var cfg Config

// Reads the configuration specified in the configuration file.
func readConfig() {
	f, err := os.Open("config/config.yml")
	if err != nil {
		log.Fatal("Could not read config file")
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatal("Could not parse config file")
	}
}
