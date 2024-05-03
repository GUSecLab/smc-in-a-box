package config

import (
	"encoding/json"
	"log"
	"os"
)

type Client struct {
	Client_ID string
	Token     string
	URLs      []string //server url
	N         int
	T         int
	K         int
	Q         int
	N_secrets int
	M         int
	N_open    int
}

func NewConfig() *Client {
	return &Client{}
}

func Load(path string) *Client {
	file, err := os.Open(path)
	if err != nil {
		log.Printf("error opening the JSON file\n")
		log.Fatalf("%s", err)
		return nil
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	config := Client{}
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatalf("unable to read from client config file: %s", err)
		return nil
	}
	log.Printf("finished reading config file \n")
	return &config
}
