package services

import (
	"bytes"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"time"
)

type DbConfig struct {
	bucket *storage.BucketHandle
	Config Config
}

type Cluster struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

type Services struct {
	Url      string `json:"url"`
	Name     string `json:"name"`
	Versions []time.Time
}

type Config struct {
	Version    string     `json:"version"`
	LastUpdate time.Time  `json:"last_update"`
	Cluster    Cluster    `json:"cluster"`
	Services   []Services `json:"services"`
}

func (self *DbConfig) Init(projectId string) {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	bucketName := projectId + "-config"
	self.bucket = storageClient.Bucket(bucketName)
	_, err = self.bucket.Attrs(ctx)

	if err != nil {
		log.Fatal("Bucket for config doesnt exist")
	}
}

func (self *DbConfig) Load() {
	ctx := context.Background()
	rc, err := self.bucket.Object("config.json").NewReader(ctx)
	defer rc.Close()
	if err != nil {
		log.Fatal("Unable to load the config file")
		return
	}
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		log.Fatal("Unable to load the config file")
		return
	}
	err = json.Unmarshal(data, &self.Config)
	if err != nil {
		log.Fatal("Unable to load the config file")
		return
	}
}

func (self *DbConfig) Save() {
	ctx := context.Background()
	wr := self.bucket.Object("config.json").NewWriter(ctx)
	defer wr.Close()
	self.Config.LastUpdate = time.Now()
	b, err := json.Marshal(self.Config)
	if err != nil {
		fmt.Println(err)
	}
	if _, err := io.Copy(wr, bytes.NewReader(b)); err != nil {
		log.Fatal(err)
	}
}

func (self *DbConfig) Initialize() {
	ctx := context.Background()
	wr := self.bucket.Object("config.json").NewWriter(ctx)
	defer wr.Close()

	config := Config{
		Version:    "0.0.0",
		LastUpdate: time.Now(),
	}
	b, err := json.Marshal(config)
	if err != nil {
		fmt.Println(err)
	}
	if _, err := io.Copy(wr, bytes.NewReader(b)); err != nil {
		log.Fatal(err)
	}
}

func GetConfig(projectId string) (DbConfig, error) {
	dbc := DbConfig{}
	dbc.Init(projectId)
	dbc.Load()
	return dbc, nil
}
