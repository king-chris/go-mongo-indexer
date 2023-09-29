package main

import "go.mongodb.org/mongo-driver/bson"

type ConfigCollection struct {
	Collection string `json:"collection"`
	Indexes    []IndexInfo
}

type IndexDiff struct {
	Old map[string]map[string]IndexInfo
	New map[string]map[string]IndexInfo
}

type IndexInfo struct {
	Key                     bson.D                 `bson:"key"`
	Name                    string                 `bson:"name"`
	Unique                  bool                   `bson:"unique,omitempty"`
	Weights                 map[string]interface{} `bson:"weights,omitempty"`
	PartialFilterExpression map[string]interface{} `bson:"partialFilterExpression,omitempty"`
	Collation               map[string]interface{} `bson:"collation,omitempty"`
}
