# go-mongo-indexer

> CLI utility to manage mongodb collection indexes

## Usage

```shell
indexer --config <index-config-file> 
        --uri <mongodb-connection-uri>
        --database <database name>
        --apply
        --fetch
```

Details of options is listed below

| **Option** | **Required?** | **Description**                                                                                              |
|------------|---------------|--------------------------------------------------------------------------------------------------------------|
| `config`   | Yes           | Path to [indexes configuration file](#config-format)                                                         |
| `uri`      | Yes           | MongoDB connection string e.g. `mongodb://127.0.0.1:27017`                                                   |
| `database` | Yes           | Database name                                                                                                |
| `apply`    | No            | Whether to apply the indexes on collections or not. If not given, it will show the plan that will be applied |
| `fetch`    | No            | Retrieves existing indexes and writes them to the specified config file. |


## Config Format

The configuration file is just a simple json file containing the indexes to be applied. This file is an array of objects. Where each object has details like collection name, cap size and indexes for this specific collection.
```javascript
[
  {
    "collection": "mongockLock",
    "indexes": [
      {
        "key": [
          {
            "key": "key",
            "value": 1
          }
        ],
        "name": "key_1",
        "collationOptions": {
          
        }
      }
    ]
  }
]
```

## Examples

> See list of index changes before applying

```shell
indexer --config "/path/to/xyz.json" --uri "mongodb://127.0.0.1:27017/database_name" --database "database_name"
```

<p align="center">
        <img src="https://i.imgur.com/3yj4gMh.png" height="400px"/>
</p>

> Apply the index changes
```shell
$ indexer --config "/path/to/xyz.json" --uri "mongodb://127.0.0.1:27017/database_name"  --database "database_name" --apply
```