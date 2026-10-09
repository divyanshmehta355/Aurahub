package opensearch

const IndexVideos = "aurahub_videos"

// IndexVideosSchema defines the index settings, tokenizers, analyzers, and mapping
// for full-text BM25 search and edge-ngram autocomplete in OpenSearch.
const IndexVideosSchema = `{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 1,
    "analysis": {
      "tokenizer": {
        "edge_ngram_tokenizer": {
          "type": "edge_ngram",
          "min_gram": 2,
          "max_gram": 20,
          "token_chars": ["letter", "digit"]
        }
      },
      "analyzer": {
        "autocomplete_analyzer": {
          "type": "custom",
          "tokenizer": "edge_ngram_tokenizer",
          "filter": ["lowercase"]
        },
        "autocomplete_search": {
          "type": "custom",
          "tokenizer": "standard",
          "filter": ["lowercase"]
        }
      }
    }
  },
  "mappings": {
    "properties": {
      "id": { "type": "keyword" },
      "fileId": { "type": "keyword" },
      "title": {
        "type": "text",
        "analyzer": "standard",
        "fields": {
          "autocomplete": {
            "type": "text",
            "analyzer": "autocomplete_analyzer",
            "search_analyzer": "autocomplete_search"
          },
          "keyword": {
            "type": "keyword",
            "ignore_above": 256
          }
        }
      },
      "description": {
        "type": "text",
        "analyzer": "standard"
      },
      "category": { "type": "keyword" },
      "tags": { "type": "keyword" },
      "thumbnailUrl": {
        "type": "keyword",
        "index": false
      },
      "views": { "type": "integer" },
      "isShort": { "type": "boolean" },
      "isAdult": { "type": "boolean" },
      "visibility": { "type": "keyword" },
      "uploaderId": { "type": "keyword" },
      "uploaderUsername": {
        "type": "text",
        "fields": {
          "keyword": { "type": "keyword" }
        }
      },
      "createdAt": { "type": "date" }
    }
  }
}`
