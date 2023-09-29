package main

type IndexDiff struct {
	Old map[string]map[string]IndexInfo
	New map[string]map[string]IndexInfo
}

// Update IndexInfo to match the new structure for keys
type IndexInfo struct {
	Key              []IndexKey       `json:"key"`
	Name             string           `json:"name"`
	CollationOptions CollationOptions `json:"collationOptions,omitempty"`
}

// New type for representing key-value pairs in an index
type IndexKey struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type ConfigCollection struct {
	Collection string      `json:"collection"`
	Indexes    []IndexInfo `json:"indexes"`
}
type CollationOptions struct {
	Locale          string `json:"locale,omitempty"`
	NumericOrdering bool   `json:"numericOrdering,omitempty"`
}
