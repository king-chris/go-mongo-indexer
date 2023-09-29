package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	jsoniter "github.com/json-iterator/go"
)

func ConfigCollections() []ConfigCollection {

	path, _ := filepath.Abs(*config)

	jsonFile, err := os.Open(path)

	if err != nil {
		log.Fatalln(err.Error())
	}

	defer jsonFile.Close()

	content, _ := ioutil.ReadAll(jsonFile)

	var collections []ConfigCollection

	json.Unmarshal(content, &collections)

	return collections
}

func GetConfigCollection(collection string) *ConfigCollection {
	collections := ConfigCollections()

	for _, c := range collections {
		if c.Collection == collection {
			return &c
		}
	}

	return nil
}

func writeConfigToFile(configCollections []ConfigCollection) error {
	var json = jsoniter.ConfigCompatibleWithStandardLibrary
	data, err := json.MarshalIndent(configCollections, "", "  ")
	if err != nil {
		return err
	}

	path, _ := filepath.Abs(*config)

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return err
	}
	return nil
}
