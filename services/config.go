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

type Version struct {
	Value      string    `json:"value"`
	LastDeploy time.Time `json:"last_deploy"`
}

type Services struct {
	Url      string    `json:"url"`
	Name     string    `json:"name"`
	Versions []Version `json:"versions"`
}

type Config struct {
	Version    string      `json:"version"`
	LastUpdate time.Time   `json:"last_update"`
	Services   []*Services `json:"services"`
}

func (self *DbConfig) Init(projectId string) error {
	ctx := context.Background()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}
	bucketName := projectId + "-config"
	self.bucket = storageClient.Bucket(bucketName)
	_, err = self.bucket.Attrs(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (self *DbConfig) Load() error {
	ctx := context.Background()
	rc, err := self.bucket.Object("config.json").NewReader(ctx)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, &self.Config)
	if err != nil {
		return err
	}
	rc.Close()
	return nil
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

func (self *DbConfig) Initialize() error {
	ctx := context.Background()
	wr := self.bucket.Object("config.json").NewWriter(ctx)
	defer wr.Close()

	config := Config{
		Version:    "0.0.0",
		LastUpdate: time.Now(),
	}
	b, err := json.Marshal(config)
	if err != nil {
		return err
	}
	if _, err := io.Copy(wr, bytes.NewReader(b)); err != nil {
		return err
	}
	return nil
}

func GetConfig(projectId string) (DbConfig, error) {
	dbc := DbConfig{}
	dbc.Init(projectId)
	dbc.Load()
	return dbc, nil
}
