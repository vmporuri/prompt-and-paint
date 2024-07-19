package main

import (
	"encoding/json"
	"log"
	"os"
)

// Holds the configuration information for the Go server.
type Config struct {
	Server struct {
		Port string `json:"port"`
		Host string `json:"host"`
	} `json:"server"`
	Database struct {
		RedisHost string `json:"redisHost"`
		RedisPort string `json:"redisPort"`
	} `json:"database"`
	Security struct {
		AllowedOrigins []string `json:"allowedOrigins"`
	} `json:"security"`
}

var cfg Config

// Reads the configuration specified in the configuration file.
func readConfig() {
	f, err := os.Open("config/config.json")
	if err != nil {
		log.Fatalf("Error opening config file: %v", err)
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		log.Fatalf("Error parsing config file: %v", err)
	}
}
