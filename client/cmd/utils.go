package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"log"
	"os"

	"example.com/SMC/pkg/ligero"
)

type ClientRequest struct {
	Exp_ID    string       `json:"Exp_ID"`
	Client_ID string       `json:"Client_ID"`
	Token     string       `json:"Token"`
	Timestamp string       `json:"Timestamp"`
	Proof     ligero.Proof `json:"Proof"`
}

type Input struct {
	Exp_ID  string `json:"Exp_ID"`
	Secrets []int  `json:"Secrets"`
}

func (c *ClientRequest) ToJson() []byte {
	msg := &ClientRequest{
		Exp_ID:    c.Exp_ID,
		Client_ID: c.Client_ID,
		Proof:     c.Proof,
		Timestamp: c.Timestamp,
	}
	message, err := json.Marshal(msg)

	if err != nil {
		log.Fatalf("Cannot marshall client request: %s", err)
	}

	// Compress the JSON data using Gzip
	var compressedData bytes.Buffer
	gzipWriter := gzip.NewWriter(&compressedData)
	_, err = gzipWriter.Write(message)
	if err != nil {
		log.Fatalf("Cannot compress client request: %s", err)
	}
	if err := gzipWriter.Close(); err != nil {
		log.Fatal(err)
	}

	return compressedData.Bytes()
}

func ReadClientInput(path string) []Input {
	jsonData, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("%s", err)
		return nil
	}

	var items []Input
	err = json.Unmarshal(jsonData, &items)
	if err != nil {
		log.Fatalf("%s", err)
		return nil
	}
	return items

}
