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
		collectionNames, err := db.ListCollectionNames(context.TODO(), bson.M{})
		if err != nil {
			log.Fatal(err)
		}

		var configCollections []ConfigCollection
		for _, collectionName := range collectionNames {
			indexes, err := getIndexesForCollection(collectionName)
			if err != nil {
				log.Fatal(err)
			}
			configCollections = append(configCollections, ConfigCollection{
				Collection: collectionName,
				Indexes:    indexes,
			})
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
func DropIndex(collection string, indexName string) bool {
	indexes := db.Collection(collection).Indexes()
	_, err := indexes.DropOne(context.TODO(), indexName)

	if err != nil {
		log.Fatalln(err.Error())
	}

	return true
}

// Create index of on the given collection with index Name and columns
func CreateIndex(collection string, indexName string, index IndexInfo) error {
	coll := db.Collection(collection)
	indexOptions := options.Index().SetName(index.Name)

	// Transform the IndexKey slice to a BSON document
	keysDoc := bson.D{}
	for _, key := range index.Key {
		keysDoc = append(keysDoc, bson.E{Key: key.Key, Value: key.Value})
	}

	// Set collation if specified and non-empty
	if len(index.CollationOptions.Locale) > 0 {
		collation := options.Collation{
			Locale:          index.CollationOptions.Locale,
			NumericOrdering: index.CollationOptions.NumericOrdering,
		}
		indexOptions.SetCollation(&collation)
	}

	indexModel := mongo.IndexModel{
		Keys:    keysDoc, // Updated to use keysDoc
		Options: indexOptions,
	}

	_, err := coll.Indexes().CreateOne(context.Background(), indexModel)
	if err != nil {
		log.Printf("Error creating index: %v", err)
	}
	return err
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
			indexJson, err := json.MarshalIndent(index, "", "  ")
			if err != nil {
				log.Printf("Error encoding index to JSON: %v", err)
				continue
			}
			util.PrintRed(fmt.Sprintf("- %s: %s\n", index.Name, string(indexJson)))
		}

		for _, index := range indexesToAdd {
			indexJson, err := json.MarshalIndent(index, "", "  ")
			if err != nil {
				log.Printf("Error encoding index to JSON: %v", err)
				continue
			}
			util.PrintGreen(fmt.Sprintf("+ %s: %s\n", index.Name, string(indexJson)))
		}
	}
}

// Generate a unique hash for an IndexInfo based on all its attributes
func hashIndex(index IndexInfo) string {
	content, _ := json.Marshal(index) // Note: consider handling the error
	algorithm := md5.New()
	algorithm.Write(content)
	return hex.EncodeToString(algorithm.Sum(nil))
}

func getIndexesDiff() *IndexDiff {
	oldIndexes := make(map[string]map[string]IndexInfo)
	newIndexes := make(map[string]map[string]IndexInfo)

	configCollections := ConfigCollections()

	for _, configCollection := range configCollections {
		collectionName := configCollection.Collection
		currentIndexes, err := getIndexesForCollection(collectionName)
		if err != nil {
			log.Printf("Error while getting indexes for collection %s: %s", collectionName, err)
			continue // or handle the error in some other way
		}

		// Maps to store the indexes by hash for easy comparison
		configIndexMap := make(map[string]IndexInfo)
		currentIndexMap := make(map[string]IndexInfo)

		// Populate the maps with the indexes from the config file and the database
		for _, index := range configCollection.Indexes {
			configIndexMap[hashIndex(index)] = index
		}
		for _, index := range currentIndexes {
			currentIndexMap[hashIndex(index)] = index
		}

		// Compare the indexes and populate oldIndexes and newIndexes
		for hash, index := range configIndexMap {
			if _, exists := currentIndexMap[hash]; !exists {
				// This index is in the config file but not in the database, or has different configuration
				if newIndexes[collectionName] == nil {
					newIndexes[collectionName] = make(map[string]IndexInfo)
				}
				newIndexes[collectionName][index.Name] = index
			}
		}
		for hash, index := range currentIndexMap {
			if _, exists := configIndexMap[hash]; !exists {
				// This index is in the database but not in the config file, or has different configuration
				if oldIndexes[collectionName] == nil {
					oldIndexes[collectionName] = make(map[string]IndexInfo)
				}
				oldIndexes[collectionName][index.Name] = index
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

// Drop index from collection by index Name
func IsCollectionToIndex(collection string) bool {
	return GetConfigCollection(collection) != nil
}

func getIndexesForCollection(collectionName string) ([]IndexInfo, error) {
	collection := db.Collection(collectionName)
	indexesCursor, err := collection.Indexes().List(context.TODO())
	if err != nil {
		return nil, err
	}

	var indexes []IndexInfo
	for indexesCursor.Next(context.TODO()) {
		var indexData bson.D
		if err := indexesCursor.Decode(&indexData); err != nil {
			return nil, err
		}

		index := IndexInfo{}

		for _, item := range indexData {
			key := item.Key
			value := item.Value

			if key == "key" {
				keyValuePairs := value.(bson.D)
				for _, keyValuePair := range keyValuePairs {
					index.Key = append(index.Key, IndexKey{Key: keyValuePair.Key, Value: keyValuePair.Value})
				}
			} else if key == "name" {
				index.Name = value.(string)
			} else if key == "collation" {
				collationMap := value.(bson.D)
				for _, collationItem := range collationMap {
					if collationItem.Key == "locale" {
						index.CollationOptions.Locale = collationItem.Value.(string)
					} else if collationItem.Key == "numericOrdering" {
						index.CollationOptions.NumericOrdering = collationItem.Value.(bool)
					}
				}
			}
		}

		if index.Name != "_id_" {
			indexes = append(indexes, index)
		}
	}
	return indexes, nil
}
