package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/idnan/go-mongo-indexer/pkg/util"
)

const GB1 = 1000000000

// Execute the command
func execute() {
	if *fetch {
		configCollections, err := fetchIndexes()
		if err != nil {
			log.Fatal(err)
		}

		err = writeConfigToFile(configCollections)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	indexDiff := getIndexesDiff()

	if !*apply {
		showDiff(indexDiff)
	}

	if *apply {
		applyDiff(indexDiff)
	}
}

func fetchIndexes() ([]ConfigCollection, error) {
	collectionNames, err := db.ListCollectionNames(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}

	var configCollections []ConfigCollection
	for _, collectionName := range collectionNames {
		collection := db.Collection(collectionName)
		indexesCursor, err := collection.Indexes().List(context.TODO())
		if err != nil {
			return nil, err
		}

		var indexes []IndexInfo
		for indexesCursor.Next(context.TODO()) {
			var index IndexInfo
			if err := indexesCursor.Decode(&index); err != nil {
				return nil, err
			}
			if index.Name != "_id_" { // Exclude the default _id index
				indexes = append(indexes, index)
			}
		}

		configCollections = append(configCollections, ConfigCollection{
			Collection: collectionName,
			Indexes:    indexes,
		})
	}

	return configCollections, nil
}

// Drop and apply the indexes
func applyDiff(indexDiff *IndexDiff) {
	for _, collection := range Collections() {
		indexesToRemove := indexDiff.Old[collection]
		indexesToAdd := indexDiff.New[collection]

		util.PrintBold(fmt.Sprintf("\n%s.%s\n", db.Name(), collection))

		if indexesToRemove == nil && indexesToAdd == nil {
			util.PrintGreen(fmt.Sprintln("No index changes"))
			continue
		}

		for _, index := range indexesToRemove {
			util.PrintRed(fmt.Sprintf("- Dropping index %s: %s\n", index.Name, util.JsonEncode(index.Key)))
			DropIndex(collection, index.Name)
		}

		for _, index := range indexesToAdd {
			util.PrintGreen(fmt.Sprintf("+ Adding index %s: %s\n", index.Name, util.JsonEncode(index.Key)))
			CreateIndex(collection, index.Name, index)
		}
	}
}

// Create index of on the given collection with index Name and columns
func CreateIndex(collection string, indexName string, index IndexInfo) error {
	coll := db.Collection(collection)
	indexOptions := options.Index().SetName(index.Name).SetUnique(index.Unique)

	if index.Weights != nil {
		indexOptions.SetWeights(index.Weights)
	}

	if index.PartialFilterExpression != nil {
		indexOptions.SetPartialFilterExpression(index.PartialFilterExpression)
	}

	if index.Collation != nil {
		collation := options.Collation{}
		// Assume the collation map contains the locale field.
		if locale, ok := index.Collation["locale"].(string); ok {
			collation.Locale = locale
		}
		indexOptions.SetCollation(&collation)
	}

	indexModel := mongo.IndexModel{
		Keys:    index.Key,
		Options: indexOptions,
	}

	_, err := coll.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Printf("Error creating index: %v", err)
	}
	return err
}

// Drop an index by Name from given collection
func DropIndex(collection string, indexName string) bool {
	indexes := db.Collection(collection).Indexes()
	_, err := indexes.DropOne(context.TODO(), indexName)

	if err != nil {
		log.Fatalln(err.Error())
	}

	return true
}

// Show the index difference, the indexes with `-` will be deleted only
// the ones with the `+` will be created
func showDiff(indexDiff *IndexDiff) {

	for _, collection := range Collections() {
		indexesToRemove := indexDiff.Old[collection]
		indexesToAdd := indexDiff.New[collection]

		util.PrintBold(fmt.Sprintf("\n%s.%s\n", db.Name(), collection))

		if indexesToRemove == nil && indexesToAdd == nil {
			util.PrintGreen(fmt.Sprintln("No index changes"))
			continue
		}

		for _, index := range indexesToRemove {
			util.PrintRed(fmt.Sprintf("- %s: %s\n", index.Name, util.JsonEncode(index.Key)))
		}

		for _, index := range indexesToAdd {
			util.PrintGreen(fmt.Sprintf("+ %s: %s\n", index.Name, util.JsonEncode(index.Key)))
		}
	}
}

// Match existing indexes with the given config file and match and find the diff
// the indexes that are not inside the config will be deleted, only the indexes in
// the config file will be created
func getIndexesDiff() *IndexDiff {
	oldIndexes := make(map[string]map[string]IndexInfo)
	newIndexes := make(map[string]map[string]IndexInfo)

	configCollections := ConfigCollections() // Assuming this function returns the collections from the config file

	for _, configCollection := range configCollections {
		collectionName := configCollection.Collection
		currentIndexes := DbIndexes(collectionName) // Assuming this function returns the indexes from the database

		// Maps to store the indexes by name for easy comparison
		configIndexMap := make(map[string]IndexInfo)
		currentIndexMap := make(map[string]IndexInfo)

		// Populate the maps with the indexes from the config file and the database
		for _, index := range configCollection.Indexes {
			configIndexMap[index.Name] = index
		}
		for _, index := range currentIndexes {
			currentIndexMap[index.Name] = index
		}

		// Compare the indexes and populate oldIndexes and newIndexes
		for name, index := range configIndexMap {
			if _, exists := currentIndexMap[name]; !exists {
				// This index is in the config file but not in the database, so it needs to be added
				if newIndexes[collectionName] == nil {
					newIndexes[collectionName] = make(map[string]IndexInfo)
				}
				newIndexes[collectionName][name] = index
			}
		}
		for name, index := range currentIndexMap {
			if _, exists := configIndexMap[name]; !exists {
				// This index is in the database but not in the config file, so it needs to be removed
				if oldIndexes[collectionName] == nil {
					oldIndexes[collectionName] = make(map[string]IndexInfo)
				}
				oldIndexes[collectionName][name] = index
			}
		}
	}

	return &IndexDiff{Old: oldIndexes, New: newIndexes}
}

// Generate index Name by doing md5 of indexes json
func GenerateIndexName(indexColumns interface{}) string {
	content, _ := json.Marshal(indexColumns)
	algorithm := md5.New()
	algorithm.Write(content)

	return hex.EncodeToString(algorithm.Sum(nil))
}

// Return list of database collections
func Collections() []string {
	collections, err := db.ListCollectionNames(context.TODO(), bson.M{})

	if err != nil {
		log.Fatalln(err.Error())
	}

	return collections
}

func DbIndexes(collection string) []IndexInfo {
	coll := db.Collection(collection)
	cursor, err := coll.Indexes().List(context.Background())
	if err != nil {
		log.Printf("Error while listing indexes: %s", err)
		return nil
	}
	var indexes []IndexInfo
	for cursor.Next(context.Background()) {
		var index IndexInfo
		if err = cursor.Decode(&index); err != nil {
			log.Printf("Error while decoding index: %s", err)
			return nil
		}
		if index.Name == "_id_" {
			continue
		}
		indexes = append(indexes, index)
	}
	if err = cursor.Err(); err != nil {
		log.Printf("Error after iterating through indexes: %s", err)
		return nil
	}
	cursor.Close(context.Background())
	return indexes
}

// Drop index from collection by index Name
func IsCollectionToIndex(collection string) bool {
	return GetConfigCollection(collection) != nil
}
