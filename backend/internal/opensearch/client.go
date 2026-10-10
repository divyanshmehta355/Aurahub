package opensearch

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	osClient "github.com/opensearch-project/opensearch-go/v5"
	"github.com/opensearch-project/opensearch-go/v5/opensearchapi"
)

type Config struct {
	Addresses          []string
	Username           string
	Password           string
	InsecureSkipVerify bool
}

type Client struct {
	api     *opensearchapi.Client
	enabled bool
}

// NewClient initializes the OpenSearch client. If no addresses are provided,
// it returns a standby client that safely reports Enabled() == false.
func NewClient(cfg Config) (*Client, error) {
	if len(cfg.Addresses) == 0 || strings.TrimSpace(cfg.Addresses[0]) == "" {
		log.Println("[OpenSearch] No cluster URL configured. Client operating in standby mode.")
		return &Client{enabled: false}, nil
	}

	cleanedAddresses := make([]string, 0, len(cfg.Addresses))
	username := cfg.Username
	password := cfg.Password

	for _, addr := range cfg.Addresses {
		trimmed := strings.TrimSpace(addr)
		if trimmed == "" {
			continue
		}

		u, err := url.Parse(trimmed)
		if err == nil && u.User != nil {
			if username == "" {
				username = u.User.Username()
			}
			if password == "" {
				if p, ok := u.User.Password(); ok {
					password = p
				}
			}
			u.User = nil
			cleanedAddresses = append(cleanedAddresses, u.String())
		} else {
			cleanedAddresses = append(cleanedAddresses, trimmed)
		}
	}

	if len(cleanedAddresses) == 0 {
		log.Println("[OpenSearch] No valid cluster URL configured. Client operating in standby mode.")
		return &Client{enabled: false}, nil
	}

	tp := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.InsecureSkipVerify {
		tp.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	// Disable client node discovery on managed cloud clusters (e.g. Aiven, AWS, etc.).
	// Cloud nodes publish their internal IP addresses which lack IP SANs on
	// cloud-issued TLS certificates (*.aivencloud.com), causing x509 validation failures.
	discoverOnStart := false

	osCfg := opensearchapi.Config{
		Client: osClient.Config{
			Addresses:             cleanedAddresses,
			Username:              username,
			Password:              password,
			Transport:             tp,
			InsecureSkipVerify:    cfg.InsecureSkipVerify,
			DiscoverNodesOnStart:  &discoverOnStart,
			DiscoverNodesInterval: 0,
		},
	}

	apiClient, err := opensearchapi.NewClient(osCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create opensearch client: %w", err)
	}

	log.Printf("[OpenSearch] Client initialized for cluster: %v\n", cleanedAddresses)
	return &Client{
		api:     apiClient,
		enabled: true,
	}, nil
}

func (c *Client) Enabled() bool {
	return c != nil && c.enabled && c.api != nil
}

// Ping checks connectivity to the OpenSearch cluster.
func (c *Client) Ping(ctx context.Context) error {
	if !c.Enabled() {
		return fmt.Errorf("opensearch client is disabled")
	}
	_, err := c.api.Ping(ctx, &opensearchapi.PingReq{})
	return err
}

// EnsureIndex checks if the aurahub_videos index exists, creating it with analyzers and mappings if not.
// It retries with backoff to handle container startup orchestration smoothly.
func (c *Client) EnsureIndex(ctx context.Context) error {
	if !c.Enabled() {
		return nil
	}

	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		existsResp, err := c.api.Indices.Exists(ctx, &opensearchapi.IndicesExistsReq{
			Indices: []string{IndexVideos},
		})
		if err == nil {
			if existsResp.StatusCode == http.StatusOK {
				log.Printf("[OpenSearch] Index %q verified.\n", IndexVideos)
				return nil
			}

			log.Printf("[OpenSearch] Index %q not found, creating with edge-ngram schema...\n", IndexVideos)
			createResp, err := c.api.Indices.Create(ctx, opensearchapi.IndicesCreateReq{
				Index:      IndexVideos,
				BodyReader: strings.NewReader(IndexVideosSchema),
			})
			if err != nil {
				return fmt.Errorf("failed to create index %q: %w", IndexVideos, err)
			}
			if !createResp.Acknowledged {
				log.Printf("[OpenSearch] Warning: index creation acknowledgment was false\n")
			} else {
				log.Printf("[OpenSearch] Index %q successfully created.\n", IndexVideos)
			}
			return nil
		}

		lastErr = err
		if attempt < 10 {
			time.Sleep(2 * time.Second)
		}
	}

	return fmt.Errorf("failed to ensure index %q after retries: %w", IndexVideos, lastErr)
}

// IndexVideo indexes or updates a video document in OpenSearch.
func (c *Client) IndexVideo(ctx context.Context, doc VideoDocument) error {
	if !c.Enabled() {
		return nil
	}

	data, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("failed to marshal video doc: %w", err)
	}

	_, err = c.api.Document.Index(ctx, opensearchapi.IndexReq{
		Index: IndexVideos,
		ID:    doc.ID,
		Body:  bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("opensearch index error for video %s: %w", doc.ID, err)
	}
	return nil
}

// BulkIndexVideos indexes multiple videos in a single bulk request.
func (c *Client) BulkIndexVideos(ctx context.Context, docs []VideoDocument) (int, error) {
	if !c.Enabled() || len(docs) == 0 {
		return 0, nil
	}

	var buf bytes.Buffer
	for _, doc := range docs {
		meta := fmt.Sprintf(`{"index":{"_index":"%s","_id":"%s"}}`+"\n", IndexVideos, doc.ID)
		buf.WriteString(meta)
		docBytes, err := json.Marshal(doc)
		if err != nil {
			continue
		}
		buf.Write(docBytes)
		buf.WriteByte('\n')
	}

	resp, err := c.api.Bulk(ctx, opensearchapi.BulkReq{
		Body: &buf,
	})
	if err != nil {
		return 0, fmt.Errorf("bulk index error: %w", err)
	}

	indexed := len(docs)
	if resp != nil && resp.Errors {
		log.Printf("[OpenSearch] Bulk indexing completed with some item errors\n")
	}
	return indexed, nil
}

// DeleteVideo removes a video document from OpenSearch by its ID.
func (c *Client) DeleteVideo(ctx context.Context, videoID string) error {
	if !c.Enabled() || videoID == "" {
		return nil
	}

	_, err := c.api.Document.Delete(ctx, opensearchapi.DeleteReq{
		Index: IndexVideos,
		ID:    videoID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete video %s from opensearch: %w", videoID, err)
	}
	return nil
}

// SearchVideos executes a BM25 ranked full-text query with field boosting and faceted filtering.
func (c *Client) SearchVideos(ctx context.Context, q string, category string, showAdult bool, sort string, limit, offset int) (*SearchResult, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("opensearch is disabled")
	}

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	filterClauses := []map[string]any{
		{"term": map[string]any{"visibility": "public"}},
	}
	if !showAdult {
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{"isAdult": false},
		})
	}
	if category != "" && category != "All" {
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{"category": category},
		})
	}

	var queryClause map[string]any
	cleanQ := strings.TrimSpace(q)
	if cleanQ == "" {
		queryClause = map[string]any{"match_all": map[string]any{}}
	} else {
		queryClause = map[string]any{
			"bool": map[string]any{
				"should": []map[string]any{
					{
						"match_phrase": map[string]any{
							"title": map[string]any{
								"query": cleanQ,
								"boost": 10,
							},
						},
					},
					{
						"multi_match": map[string]any{
							"query": cleanQ,
							"fields": []string{
								"title^5",
								"title.autocomplete^4",
								"tags^3",
								"uploaderUsername^2",
								"description^1",
							},
							"fuzziness":     "AUTO",
							"prefix_length": 0,
						},
					},
				},
				"minimum_should_match": 1,
			},
		}
	}

	var sortClause []any
	switch strings.ToLower(sort) {
	case "latest", "recent":
		sortClause = []any{map[string]any{"createdAt": "desc"}}
	case "trending", "popular":
		sortClause = []any{
			map[string]any{"views": "desc"},
			map[string]any{"_score": "desc"},
		}
	default: // "relevance"
		sortClause = []any{
			map[string]any{"_score": "desc"},
			map[string]any{"views": "desc"},
		}
	}

	searchBody := map[string]any{
		"from": offset,
		"size": limit,
		"query": map[string]any{
			"bool": map[string]any{
				"must":   queryClause,
				"filter": filterClauses,
			},
		},
		"sort": sortClause,
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, err
	}

	searchResp, err := c.api.Search(ctx, &opensearchapi.SearchReq{
		Indices:    []string{IndexVideos},
		BodyReader: bytes.NewReader(bodyBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch search failed: %w", err)
	}

	var totalHits int64
	if searchResp.Hits.Total != nil {
		if th, err := searchResp.Hits.Total.TotalHits(); err == nil {
			totalHits = th.Value
		} else if val, err := searchResp.Hits.Total.Int64(); err == nil {
			totalHits = val
		}
	}

	videos := make([]VideoDocument, 0, len(searchResp.Hits.Hits))
	for _, hit := range searchResp.Hits.Hits {
		var doc VideoDocument
		if err := json.Unmarshal(hit.Source, &doc); err == nil {
			videos = append(videos, doc)
		}
	}

	return &SearchResult{
		Videos: videos,
		Total:  totalHits,
	}, nil
}

// Autocomplete performs an edge-ngram prefix search returning title and video suggestions.
func (c *Client) Autocomplete(ctx context.Context, prefix string, showAdult bool, limit int) ([]AutocompleteSuggestion, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("opensearch is disabled")
	}

	cleanPrefix := strings.TrimSpace(prefix)
	if len([]rune(cleanPrefix)) < 2 {
		return []AutocompleteSuggestion{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	filterClauses := []map[string]any{
		{"term": map[string]any{"visibility": "public"}},
	}
	if !showAdult {
		filterClauses = append(filterClauses, map[string]any{
			"term": map[string]any{"isAdult": false},
		})
	}

	searchBody := map[string]any{
		"size": limit,
		"query": map[string]any{
			"bool": map[string]any{
				"should": []map[string]any{
					{
						"multi_match": map[string]any{
							"query": cleanPrefix,
							"fields": []string{
								"title.autocomplete^5",
								"title^3",
							},
						},
					},
					{
						"multi_match": map[string]any{
							"query": cleanPrefix,
							"fields": []string{
								"title^3",
								"title.autocomplete^2",
							},
							"fuzziness": "AUTO",
						},
					},
				},
				"minimum_should_match": 1,
				"filter":               filterClauses,
			},
		},
		"sort": []any{
			map[string]any{"_score": "desc"},
			map[string]any{"views": "desc"},
		},
	}

	bodyBytes, err := json.Marshal(searchBody)
	if err != nil {
		return nil, err
	}

	searchResp, err := c.api.Search(ctx, &opensearchapi.SearchReq{
		Indices:    []string{IndexVideos},
		BodyReader: bytes.NewReader(bodyBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch autocomplete failed: %w", err)
	}

	suggestions := make([]AutocompleteSuggestion, 0, len(searchResp.Hits.Hits))
	for _, hit := range searchResp.Hits.Hits {
		var doc VideoDocument
		if err := json.Unmarshal(hit.Source, &doc); err == nil {
			suggestions = append(suggestions, AutocompleteSuggestion{
				ID:           doc.ID,
				Title:        doc.Title,
				ThumbnailUrl: doc.ThumbnailUrl,
				Category:     doc.Category,
			})
		}
	}
	return suggestions, nil
}
