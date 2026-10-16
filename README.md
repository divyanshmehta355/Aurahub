# Aurahub 🎬

Aurahub is a modern, high-performance video streaming and creator platform built with **Go (Fiber)**, **SvelteKit**, **PostgreSQL 16**, **Valkey (Redis)**, **Apache Kafka (KRaft)**, and **OpenSearch**.

---

## 📑 Table of Contents
- [Architecture & Data Flow](#-architecture--data-flow)
  - [1. System Topology Overview](#1-system-topology-overview)
  - [2. Direct Video Upload Pipeline](#2-direct-video-upload-pipeline)
  - [3. High-Throughput View Ingestion & Batch Flush](#3-high-throughput-view-ingestion--batch-flush)
  - [4. Search & Real-Time Autocomplete Flow](#4-search--real-time-autocomplete-flow)
  - [5. Real-Time Notification & SSE Stream](#5-real-time-notification--sse-stream)
  - [6. Adult Content Safety Gating](#6-adult-content-safety-gating)
- [Key Features](#-key-features)
- [Tech Stack](#-tech-stack)
- [Zero-Config Docker Setup (Recommended)](#-zero-config-docker-setup-recommended)
  - [Service Ports & Credentials](#-service-ports--credentials)
  - [Stopping & Resetting Volumes](#-stopping--resetting-volumes)
- [Local / Hybrid Development Setup](#-local--hybrid-development-setup)
  - [Prerequisites](#prerequisites)
  - [1. Backend Setup](#1-backend-setup)
  - [2. Frontend Setup](#2-frontend-setup)
  - [3. OpenSearch Indexing & Backfill](#3-opensearch-indexing--backfill)
- [Environment Configuration (.env)](#-environment-configuration-env)
- [Project Directory Structure](#-project-directory-structure)
- [Troubleshooting & FAQs](#-troubleshooting--faqs)

---

## 🏛️ Architecture & Data Flow

### 1. System Topology Overview

[![System Topology Overview](https://mermaid.ink/svg/pako:eNp9VU1z4jgQ_SsqHyioiQM4YD4OW0UCw6QSEgaTbNWO96DYDahiLJcsExiG_74tCWcdjzMcsCx1P71uvScfrYCHYA2ttaDJhizHvvBjgr-nFMQP37qJGMSSXAv-hhOk7u0gknDHJOmSL2RJWfTG4rDhW__mifkzzV4M5pRKeKMHgmijJIlYQCXj8ft0jYzmt4V89fvKXvTuU26GZDHxliqO1If9Vr_VKMV73gSjPRA7ELanCE92-J8STwqgW1Jv0oQ1Yy7Z6rx92kz1UhEJ4vDTIjzJBV2DKmIOImWpVLvUyG1sz2DLxSGPKDGbT3_UMYWnci3A-35P2q7vxwuINAsakfE1okxvHxAphD2kvtX4APBMo1c4KBAzIk2ygJCl2An3qjdoINoNDTaAKB6kqeqsYgJFnD8XptrA4rUqTXetMFUjM8RUdaMAXkGUirujq1d6PKpz1RT0O6nfLehKIs8RQzhF0MAuecICrO90KmL8zQUCPzN4S5GAfn6NsnSDh26WML_eTcmMBYLb11QGG2RWFoAJ1S1ElMcEYg-oCDbnrr5jVWVpTMzST_KURJyGf4pfwEpAqjJGmeTntwJZp2-PUdeTfcKEkfoCYpR6VKG134_D0NZq1qMaGbM04ChshIzXLC4r7NFT4iiU7Fy2-8jjeuZ0MXsSrsF-mAp0wRKPMGY_Ub4V4viNyQxVRrUo9hKEkqqZyZ1QIzfjhxKXUSYoulS3RlBt2OZZTpImgKyeWQj8IwaZR_TwodclSuoqIrb91y_f-rZczs1lgNdGJje-9cvcEBWx9yg_8lD0vIrGq6IidswEBJIYdkYCKvhcT0XCZPsCYQgh-XbvNW9X2F4oJeRJ5gIzWd8zdYzoqkz-z2g-rQq9g4ONjs-AaHurQHMBVAXPs5eIoQjNraditRWrQs-yUNIN-DaJQGrsR6_I2RgZE4r2_GxRW-yzRW2qzxbP3inuXNgvJ2xsOR-PlhNS_7JTS42PfSvwOCeZ8ZgH2TbviKmwZPu8fzyK1AVKd1CFnHvcBGs3G3vj-T8t7tPSyVsX-DVloTWUIoMLawtiS9WrdVSgviU3oNQyxCEiZHs7OLVDnjE0QR-fML8hMb_cL7NIQTP1htruKJRim9ZEuK3c8woGnX7PivQNSBueBZLa3jluhrEGh6tvTVsu91Lt9tzHMcd9J1WS60erKHjXrqdq37P6bodp9_uDjqnC-un3rd1Oei3e_2rQavTabVxrXf6DyY6nZo)](https://mermaid.live/edit#pako:eNp9VU1z4jgQ_SsqHyioiQM4YD4OW0UCw6QSEgaTbNWO96DYDahiLJcsExiG_74tCWcdjzMcsCx1P71uvScfrYCHYA2ttaDJhizHvvBjgr-nFMQP37qJGMSSXAv-hhOk7u0gknDHJOmSL2RJWfTG4rDhW__mifkzzV4M5pRKeKMHgmijJIlYQCXj8ft0jYzmt4V89fvKXvTuU26GZDHxliqO1If9Vr_VKMV73gSjPRA7ELanCE92-J8STwqgW1Jv0oQ1Yy7Z6rx92kz1UhEJ4vDTIjzJBV2DKmIOImWpVLvUyG1sz2DLxSGPKDGbT3_UMYWnci3A-35P2q7vxwuINAsakfE1okxvHxAphD2kvtX4APBMo1c4KBAzIk2ygJCl2An3qjdoINoNDTaAKB6kqeqsYgJFnD8XptrA4rUqTXetMFUjM8RUdaMAXkGUirujq1d6PKpz1RT0O6nfLehKIs8RQzhF0MAuecICrO90KmL8zQUCPzN4S5GAfn6NsnSDh26WML_eTcmMBYLb11QGG2RWFoAJ1S1ElMcEYg-oCDbnrr5jVWVpTMzST_KURJyGf4pfwEpAqjJGmeTntwJZp2-PUdeTfcKEkfoCYpR6VKG134_D0NZq1qMaGbM04ChshIzXLC4r7NFT4iiU7Fy2-8jjeuZ0MXsSrsF-mAp0wRKPMGY_Ub4V4viNyQxVRrUo9hKEkqqZyZ1QIzfjhxKXUSYoulS3RlBt2OZZTpImgKyeWQj8IwaZR_TwodclSuoqIrb91y_f-rZczs1lgNdGJje-9cvcEBWx9yg_8lD0vIrGq6IidswEBJIYdkYCKvhcT0XCZPsCYQgh-XbvNW9X2F4oJeRJ5gIzWd8zdYzoqkz-z2g-rQq9g4ONjs-AaHurQHMBVAXPs5eIoQjNraditRWrQs-yUNIN-DaJQGrsR6_I2RgZE4r2_GxRW-yzRW2qzxbP3inuXNgvJ2xsOR-PlhNS_7JTS42PfSvwOCeZ8ZgH2TbviKmwZPu8fzyK1AVKd1CFnHvcBGs3G3vj-T8t7tPSyVsX-DVloTWUIoMLawtiS9WrdVSgviU3oNQyxCEiZHs7OLVDnjE0QR-fML8hMb_cL7NIQTP1htruKJRim9ZEuK3c8woGnX7PivQNSBueBZLa3jluhrEGh6tvTVsu91Lt9tzHMcd9J1WS60erKHjXrqdq37P6bodp9_uDjqnC-un3rd1Oei3e_2rQavTabVxrXf6DyY6nZo)

<details>
<summary><b>View Mermaid Source</b></summary>

```mermaid
graph TD
    User["Client Browser (SvelteKit 5 + Tailwind)"]
    
    subgraph Gateway ["Application Gateway & API"]
        Fiber["Go Fiber REST API (:8080)"]
        SSE["Server-Sent Events Stream (/api/notifications/stream)"]
    end
    
    subgraph Storage ["Persistent & In-Memory Storage"]
        PG[("PostgreSQL 16\nRelational DB & GIN Indexes")]
        Valkey[("Valkey / Redis (:6379)\nCache & Session Store")]
    end
    
    subgraph Streaming ["Event Streaming & Message Broker"]
        Kafka{{"Apache Kafka (KRaft / Aiven)\nEvent Topics"}}
        WorkerViews["ViewsFlusher Worker\n(5s Micro-Batching)"]
        WorkerIndex["OpenSearch Indexer Worker"]
        WorkerBatch["Batch Upload Worker"]
        WorkerRefresh["AutoRefresh Worker\n(28-Day Expiration Renewal)"]
    end

    subgraph Search ["Search & Discovery Engine"]
        OS[("OpenSearch 2.18\nBM25 & Edge-NGram Tokenizers")]
    end

    subgraph Media ["External Media Storage & CDN"]
        AuraAPI["Aura API / Streamtape\nVideo Storage & CDN Player"]
    end

    User -->|"HTTP REST & Auth"| Fiber
    User -->|"Live Notifications"| SSE
    User -->|"Direct Video Upload"| AuraAPI
    User -->|"Embedded HLS/Iframe"| AuraAPI

    Fiber -->|"Query & Mutations"| PG
    Fiber -->|"Key-Value Cache"| Valkey
    Fiber -->|"Publish Events"| Kafka
    Fiber -->|"BM25 & Autocomplete"| OS

    Kafka --> WorkerViews
    Kafka --> WorkerIndex
    Kafka --> WorkerBatch
    Kafka --> WorkerRefresh

    WorkerViews -->|"Batch UPDATE (+views)"| PG
    WorkerIndex -->|"Index Documents"| OS
    WorkerBatch -->|"Poll & Save"| PG
    WorkerRefresh -->|"Renew Expired URLs"| AuraAPI
```
</details>

---

### 2. Direct Video Upload Pipeline

Creators upload high-definition videos directly to the cloud storage target without double-buffering or chunking on the backend server, maintaining minimal server memory overhead and high upload throughput.

[![Direct Video Upload Pipeline](https://mermaid.ink/svg/pako:eNp9VNlO20AU_ZWrkVqBZCfBEJP4oVJZSmkpoUm6qIqEBs_FGWF73FkCNMq_944dh0Vp_TLbOWfOXcZLliqBLGEGfzssUzyRPNO8mOlZCfRxZ1XpihvUm53UKg3HGrkfudlMj7S6N0-4imsrU1nx0sL7q3OPPFPwQZKUX2-FOc09rh49pwsTS-qF5RVuI5wcefiVMjbTOPl6sQ3zmd_e1arNhFzebTc5mnjUqMJyglync49pcesYw3fvyFYCZ6dT6PJKdhdSoDLdDG3oqlxxETqdw86nH1MKws53Wz6xPJfiSmDsM20sGJmVKEBIjamFhg6WaxLb0IgQtpeO0Tpdgisl8V_i4dv44vlVRFk7fqI1eALC25ZlKRWbENvxUlkEtcBNjYPG9knjM80llja0KjR0xjMEU5cIFpLDz49juJd2DqZCFN1KK6qLMWA1T-9kmW1JZ619NZpMoc4l3MocqSyPtdudqSeaRi6AN9AqBmBcVSltDTzMdYff0HT3ddZep8BLX0tB8RcoyG2Blgtu-X8KXRt7XunUn2NIqVBawHItGYCVNscAUjrMlH6kDZ6RS2muuXC5Xb1shJOjBM4vJ6fjKQ3TURO6gR2v9FqyEep0Oq-aqW5ncuhucmnm8N1LXMhbTB_THE8XVCTYobcqVZnAjDW-xYzt_qtPot5eEz315LI2RC5Wz3NT30iU0SR59kzgvBT4QO2SqtK4Ag3krQtAb6OljyZrbk1Yl1uolDjktW4bFBmGpf__GKrS0ZeoD_cos7k1XoQFLNNSsMRqhwErUBfcL9nSXzBjdo4FzpgPV6NwD6Hg-i5MVa70jM3KFfHpof9SqmgltHLZnCW3PDe0chV1Q_sD3OxqJLv6WLnSsmT_sFeLsGTJHliyF_c7cf8wiqJ4OIh6vTgO2CNLorgTH-wPDqN-fBAN9vrDg1XA_tT39jrDwd7hYH8YxQeDeBgPhqu_7SzUUA)](https://mermaid.live/edit#pako:eNp9VNlO20AU_ZWrkVqBZCfBEJP4oVJZSmkpoUm6qIqEBs_FGWF73FkCNMq_944dh0Vp_TLbOWfOXcZLliqBLGEGfzssUzyRPNO8mOlZCfRxZ1XpihvUm53UKg3HGrkfudlMj7S6N0-4imsrU1nx0sL7q3OPPFPwQZKUX2-FOc09rh49pwsTS-qF5RVuI5wcefiVMjbTOPl6sQ3zmd_e1arNhFzebTc5mnjUqMJyglync49pcesYw3fvyFYCZ6dT6PJKdhdSoDLdDG3oqlxxETqdw86nH1MKws53Wz6xPJfiSmDsM20sGJmVKEBIjamFhg6WaxLb0IgQtpeO0Tpdgisl8V_i4dv44vlVRFk7fqI1eALC25ZlKRWbENvxUlkEtcBNjYPG9knjM80llja0KjR0xjMEU5cIFpLDz49juJd2DqZCFN1KK6qLMWA1T-9kmW1JZ619NZpMoc4l3MocqSyPtdudqSeaRi6AN9AqBmBcVSltDTzMdYff0HT3ddZep8BLX0tB8RcoyG2Blgtu-X8KXRt7XunUn2NIqVBawHItGYCVNscAUjrMlH6kDZ6RS2muuXC5Xb1shJOjBM4vJ6fjKQ3TURO6gR2v9FqyEep0Oq-aqW5ncuhucmnm8N1LXMhbTB_THE8XVCTYobcqVZnAjDW-xYzt_qtPot5eEz315LI2RC5Wz3NT30iU0SR59kzgvBT4QO2SqtK4Ag3krQtAb6OljyZrbk1Yl1uolDjktW4bFBmGpf__GKrS0ZeoD_cos7k1XoQFLNNSsMRqhwErUBfcL9nSXzBjdo4FzpgPV6NwD6Hg-i5MVa70jM3KFfHpof9SqmgltHLZnCW3PDe0chV1Q_sD3OxqJLv6WLnSsmT_sFeLsGTJHliyF_c7cf8wiqJ4OIh6vTgO2CNLorgTH-wPDqN-fBAN9vrDg1XA_tT39jrDwd7hYH8YxQeDeBgPhqu_7SzUUA)

<details>
<summary><b>View Mermaid Source</b></summary>

```mermaid
sequenceDiagram
    autonumber
    actor Creator as Creator Browser
    participant API as Go Fiber API
    participant Aura as Aura API / Streamtape
    participant DB as PostgreSQL
    participant Kafka as Kafka Broker
    participant OS as OpenSearch

    Creator->>API: GET /api/videos/get-upload-url (JWT Auth)
    API->>Aura: Request signed direct upload target
    Aura-->>API: Return unique upload target URL
    API-->>Creator: Return upload URL & target token
    
    Note over Creator,Aura: Direct client-to-storage stream via XHR with speed/progress tracking
    Creator->>Aura: POST video file payload (Tracks speed, % progress, supports xhr.abort)
    Aura-->>Creator: Return file_id & media metadata

    Creator->>API: POST /api/videos/create-record {file_id, title, category, tags, is_adult}
    API->>DB: INSERT INTO videos (id, file_id, title, tags, ...)
    API->>Kafka: Publish VideoLifecycleEvent (action: "created")
    API-->>Creator: 201 Created {video_id}

    Kafka->>OS: OpenSearch Indexer consumes lifecycle event
    OS->>OS: Index video document with edge-ngrams & BM25 weights
```
</details>

---

### 3. High-Throughput View Ingestion & Batch Flush

To eliminate row-level write-lock contention under viral traffic spikes, video view counts are queued into Apache Kafka and flushed to PostgreSQL in 5-second aggregated micro-batches.

[![High-Throughput View Ingestion & Batch Flush](https://mermaid.ink/svg/pako:eNqNVGtv2jAU_StX_tSpPEoKAawNiQLdWCtgJVuljakyiRsskpj5AWWI_77rBGi1ddryIX7k3HPPudfOjoQy4oQSzX9YnoW8L1isWDpTswzwYdbIzKZzrk47oZEKvgi-4QqYPs6ulNzoZ9SKKSNCsWKZge5k6IDvJVwLJHLr12A37HHJHLCYIOESwWfMKraw88paRFzim2_0m9fC76VaPivS14nVC1z3ZKZtipPi-2uR_SsXNZHaxIpPP906zBFXuCt3OiiawmQ8DaDKVqKaq9FVKqKqkwRncmWEzFgCH--Dk76RNBzkuvBMcZmV54kMlyKLIZZKWiMyDmdvoZZqSJjBBmxPwRhSxsSFAgrexQWMb2CnbRhyrSkYZfn-pVYX0Onk1UOpdp4IvUADKNRxDNYcve5y4cOoBBa75UYjUq4NS1f7I0_OgExFxeixhIA4zWIOc2bCxZ8Wj_BujGWM0YyGUNrMaBAZxqZSbSm4YvXy3W8HJd_Pz1-aSKRcAWpVW2jAlIcyizQEInzRO_cUyVBk_4rC50m_Gwyg6AlMB0GeRsO7w3heCIH7D4O7AYgIP9BD9r9yDkfTwV2AQzDOS_WA516shRFoazyC3nh0fTvsBYfc_0WzcXV7WAiNF2j7D5Lfy1pyZF0jUxGyJNmioTQVWNq8FzzKjWKdQ8VTbLM-UvEsd0hKJFYiItSdmRLB65AytyQ7B5sRs8CwGaE4VTyyT-WIqWU5lIlUMzLL9hiPN-WrlOmRAs9uvCD0kSUaV3YVYb8Pv47TrsLsXOXdJvSyWc9JCN2RJ0JrfqPiN5qe5_ntFh5t3y-RLaGeX_Hrl62m1_DrXqvWaNf3JfIzz3tRabdqzdZl2_PrLb_tt_e_AMIXjfY)](https://mermaid.live/edit#pako:eNqNVGtv2jAU_StX_tSpPEoKAawNiQLdWCtgJVuljakyiRsskpj5AWWI_77rBGi1ddryIX7k3HPPudfOjoQy4oQSzX9YnoW8L1isWDpTswzwYdbIzKZzrk47oZEKvgi-4QqYPs6ulNzoZ9SKKSNCsWKZge5k6IDvJVwLJHLr12A37HHJHLCYIOESwWfMKraw88paRFzim2_0m9fC76VaPivS14nVC1z3ZKZtipPi-2uR_SsXNZHaxIpPP906zBFXuCt3OiiawmQ8DaDKVqKaq9FVKqKqkwRncmWEzFgCH--Dk76RNBzkuvBMcZmV54kMlyKLIZZKWiMyDmdvoZZqSJjBBmxPwRhSxsSFAgrexQWMb2CnbRhyrSkYZfn-pVYX0Onk1UOpdp4IvUADKNRxDNYcve5y4cOoBBa75UYjUq4NS1f7I0_OgExFxeixhIA4zWIOc2bCxZ8Wj_BujGWM0YyGUNrMaBAZxqZSbSm4YvXy3W8HJd_Pz1-aSKRcAWpVW2jAlIcyizQEInzRO_cUyVBk_4rC50m_Gwyg6AlMB0GeRsO7w3heCIH7D4O7AYgIP9BD9r9yDkfTwV2AQzDOS_WA516shRFoazyC3nh0fTvsBYfc_0WzcXV7WAiNF2j7D5Lfy1pyZF0jUxGyJNmioTQVWNq8FzzKjWKdQ8VTbLM-UvEsd0hKJFYiItSdmRLB65AytyQ7B5sRs8CwGaE4VTyyT-WIqWU5lIlUMzLL9hiPN-WrlOmRAs9uvCD0kSUaV3YVYb8Pv47TrsLsXOXdJvSyWc9JCN2RJ0JrfqPiN5qe5_ntFh5t3y-RLaGeX_Hrl62m1_DrXqvWaNf3JfIzz3tRabdqzdZl2_PrLb_tt_e_AMIXjfY)

<details>
<summary><b>View Mermaid Source</b></summary>

```mermaid
sequenceDiagram
    autonumber
    actor Viewer as Viewer Browser
    participant API as Go Fiber API
    participant Kafka as Kafka Broker (aurahub.video.views)
    participant Worker as ViewsFlusher Consumer Worker
    participant DB as PostgreSQL

    Viewer->>API: POST /api/videos/:id/view (optional JWT)
    Note over API: Non-blocking goroutine (< 1ms latency)
    API-->>Viewer: 200 OK {success: true}

    API->>Kafka: Publish VideoViewEvent {videoId, userId, timestamp}
    Kafka->>Worker: Consume message batch
    Note over Worker: Aggregates counts in memory: viewCounts[videoId]++

    loop Every 5 Seconds Ticker
        Worker->>DB: UPDATE videos SET views = views + count WHERE id = :videoId
        Worker->>DB: INSERT INTO user_activities ON CONFLICT UPDATE
        Worker->>DB: INSERT INTO watch_history ON CONFLICT UPDATE
        Note over Worker,DB: Atomically commits batched view increments
    end
```
</details>

---

### 4. Search & Real-Time Autocomplete Flow

[![Search & Real-Time Autocomplete Flow](https://mermaid.ink/svg/pako:eNpdkl1v2jAUhv_KkS-mTgpfCYElUjsx2lWVyqCl28WWFZnkEKwmdmo7DEb732eHQNP5yl_ned_32HsSiwRJSFJJizXc3kcy4mDGDS9K_Ssi3xVKmCOV8fqwF5Hf0GpdvETkYVcwnsLZxTm4EK-pVB8j8gKjsSm7vnqADi1YR1WlHVpqEYu8yFDj5-dzA3knVBNnEpWCK66NZgfm5TJn2iIP-k3shiUoVE1_Ax6ho7ElwnS-qNxMC-R1BkxSbHGTNgejtmJbmFAdr6OIn00YB9cx6y243VOgk9EKVhu94UpTrkGVaYpKM8GV9XkpRZGIP9xIPjCdIXyAsUSqhWnh283Ttf9M1w5r418mrv_e-qTMNGtVdsGewl2Jcmedayv26DmgaaoeTYaYakyF3D322r4DCapYssKKNwM1glW4Q7SG4GhDWUaXGdps96iMvjKW7il_wgR-2Cc4bsOMpvi-WQdm-9CvFTS409UqY7yizq4XX2mWLWn8ZMjHaQgzoXQqcX53a_Jd33yDw0svNhjbbmq1kMZFQ7ABqjpY-7LHxDG_myUk1LJEh-Qoc2qXZG9LI6LXmBszoZlKTMptK6HyqRWLTMiIRPzV1BeU_xQiPyKkKNM1CVc0U2ZVFolp9yWj9leddiXyBOVYlFyTcNANKggJ92RLwqDb7vUG3tAP_GDY8_q-Q3Yk9Idtr_9p6A37rhcEbu_VIX8r0W476AZufzgY-J5nSszRPwuRNOA)](https://mermaid.live/edit#pako:eNpdkl1v2jAUhv_KkS-mTgpfCYElUjsx2lWVyqCl28WWFZnkEKwmdmo7DEb732eHQNP5yl_ned_32HsSiwRJSFJJizXc3kcy4mDGDS9K_Ssi3xVKmCOV8fqwF5Hf0GpdvETkYVcwnsLZxTm4EK-pVB8j8gKjsSm7vnqADi1YR1WlHVpqEYu8yFDj5-dzA3knVBNnEpWCK66NZgfm5TJn2iIP-k3shiUoVE1_Ax6ho7ElwnS-qNxMC-R1BkxSbHGTNgejtmJbmFAdr6OIn00YB9cx6y243VOgk9EKVhu94UpTrkGVaYpKM8GV9XkpRZGIP9xIPjCdIXyAsUSqhWnh283Ttf9M1w5r418mrv_e-qTMNGtVdsGewl2Jcmedayv26DmgaaoeTYaYakyF3D322r4DCapYssKKNwM1glW4Q7SG4GhDWUaXGdps96iMvjKW7il_wgR-2Cc4bsOMpvi-WQdm-9CvFTS409UqY7yizq4XX2mWLWn8ZMjHaQgzoXQqcX53a_Jd33yDw0svNhjbbmq1kMZFQ7ABqjpY-7LHxDG_myUk1LJEh-Qoc2qXZG9LI6LXmBszoZlKTMptK6HyqRWLTMiIRPzV1BeU_xQiPyKkKNM1CVc0U2ZVFolp9yWj9leddiXyBOVYlFyTcNANKggJ92RLwqDb7vUG3tAP_GDY8_q-Q3Yk9Idtr_9p6A37rhcEbu_VIX8r0W476AZufzgY-J5nSszRPwuRNOA)

<details>
<summary><b>View Mermaid Source</b></summary>

```mermaid
graph LR
    Input["User Search Input"] -->|"Typing (>= 2 chars)"| AC["GET /api/search/autocomplete?q="]
    Input -->|"Press Enter / Submit"| Search["GET /api/videos/search?q="]

    AC --> OS_AC["OpenSearch edge-ngram Prefix Match\n(Min 2, Max 20 chars)"]
    OS_AC -->|"Instant suggestions"| Dropdown["Title & Creator Suggestions Dropdown"]

    Search --> OS_BM25["OpenSearch Multi-Match BM25 Query\n(title^3, tags^2, category^1.5, description)"]
    
    OS_BM25 -->|"OpenSearch Available"| Results["Ranked Video Results Page"]
    OS_BM25 -.->|"If OpenSearch Offline"| PG_Fallback["Fallback: PostgreSQL\nGIN search_vector ts_rank"]
    PG_Fallback --> Results
```
</details>

---

### 5. Real-Time Notification & SSE Stream

[![Real-Time Notification & SSE Stream](https://mermaid.ink/svg/pako:eNptk21r2zAQx7_KoVcdOM6Dk3jWi0KbdaPraAOhDIZhKPY1EbElTw9p05DvvpPTuCnUbySd7n73v7_tPSt0iYwzi_88qgK_SbEyos5NroAe4Z1Wvl6i6SKF0wZmBkVYhe22fXi072mNME4WshHKwdX8NiT-0PBdEimcP0u7E08bERKPm2ujN5R8IbwRa7-MlXbySRbCSa3sl49yFn5pC9PCqf7sNKskKvdZt8XiJuTebOl-4WiGGi76opH9D336tr1q250o7_je5SVhODw0qKBBY6V1-MYutFJYBAhc3CE2vatKbrHT_eYaEcgNDo9NpUVpYStL1GTlL7lBS-tM1zUR7amMkqmkNYjD3C8raddwf6a4nQf2BmnMMPptGYFFVaIJO7drMDo2uS0PJ2iLOyk5h1n4rU14CYRDUm8Bt2d2HsW0Dsw96fi5eLgnj3dhFA6lcKRxnzNZ5oxDzuI4zllEm6DiGFL4_LdVk7NODfF6Adu5zOEaqwokOQpGqpUFqayjd1jt4Fm6NSxFuUIKFgbrN3ksYisjS8ad8RixGk0twpHtQxOSsKbUowaDpX_plcJseoWutMlZrg5UT1_JH63rE8Jov1oz_iQqSyff0Hinn6WLmtbomfbKMT4dJi2E8T17YTwbxMPhNEkn2SRLh8l4ErEd45M0TsZf0yQdj5IsGw0PEXttmw7ibJCNxuk0mabZZDAeH_4DHR04iw)](https://mermaid.live/edit#pako:eNptk21r2zAQx7_KoVcdOM6Dk3jWi0KbdaPraAOhDIZhKPY1EbElTw9p05DvvpPTuCnUbySd7n73v7_tPSt0iYwzi_88qgK_SbEyos5NroAe4Z1Wvl6i6SKF0wZmBkVYhe22fXi072mNME4WshHKwdX8NiT-0PBdEimcP0u7E08bERKPm2ujN5R8IbwRa7-MlXbySRbCSa3sl49yFn5pC9PCqf7sNKskKvdZt8XiJuTebOl-4WiGGi76opH9D336tr1q250o7_je5SVhODw0qKBBY6V1-MYutFJYBAhc3CE2vatKbrHT_eYaEcgNDo9NpUVpYStL1GTlL7lBS-tM1zUR7amMkqmkNYjD3C8raddwf6a4nQf2BmnMMPptGYFFVaIJO7drMDo2uS0PJ2iLOyk5h1n4rU14CYRDUm8Bt2d2HsW0Dsw96fi5eLgnj3dhFA6lcKRxnzNZ5oxDzuI4zllEm6DiGFL4_LdVk7NODfF6Adu5zOEaqwokOQpGqpUFqayjd1jt4Fm6NSxFuUIKFgbrN3ksYisjS8ad8RixGk0twpHtQxOSsKbUowaDpX_plcJseoWutMlZrg5UT1_JH63rE8Jov1oz_iQqSyff0Hinn6WLmtbomfbKMT4dJi2E8T17YTwbxMPhNEkn2SRLh8l4ErEd45M0TsZf0yQdj5IsGw0PEXttmw7ibJCNxuk0mabZZDAeH_4DHR04iw)

<details>
<summary><b>View Mermaid Source</b></summary>

```mermaid
sequenceDiagram
    autonumber
    actor Creator as Creator / User
    participant API as Go Fiber API
    participant Kafka as Kafka Broker (aurahub.notifications)
    actor Subscriber as Subscriber Client
    participant SSE as EventStream (/api/notifications/stream)

    Subscriber->>SSE: Open persistent SSE connection (Keep-Alive)
    Creator->>API: Uploads video / Likes / Comments
    API->>Kafka: Publish NotificationEvent {recipientId, senderId, type, videoId}
    Kafka->>API: Notifications Worker receives event
    API->>SSE: Push JSON payload: data: {"id": "...", "type": "new_video"}
    SSE-->>Subscriber: Bell icon rings instantly with badge increment
```
</details>

---

### 6. Adult Content Safety Gating

[![Adult Content Safety Gating](https://mermaid.ink/svg/pako:eNqNU21P2zAQ_isnf9q0FggNpY20SdDyMmmIl8KQRiZk4kti1bEjx1lhpf9957xANWnS8iX22c_z3D13XrPECGQRyywvc7idxzbWQN9dqQwXDzGbWeTOWKibQAW_pEADK-lyCCaf4EjUykGqeBaznzAcfoH58eMpbQl6ZSqXWVxcfwNZPfLm5mdwtka62wv1_w7WUJwiiusa7cs6Zn4Nu7CoswwrJ42uoDmK2aaHvl334NeYzTHlXmvBU4QLqi9mr3DynKhaIKV1f35yc7KdUcpVhfDhXAqBGlJrCijrJyUTSIm5-riV7d9SdxVayHnVeIGaPykUXu2r7tW6lQCpW7qOrWe85y7J59Ji4h46PlMiVbm78ie7kRRg0oa_sb63eZZjsryyJpUK1x2w28ICnZM6q7Y82r7e5V7lZtV68JgY7VC_d-cVjpQyK0qozQyuFH954snyvfH33GrSgGMusu1-_q9QazoN1llNffWSZ9xho1iVpAYX3NUWYdYhKJwoLgsqk1rK1ZakB3ZSM-raEi41DtsVoVNpC09_a7JM4aX2LpfCQ_os_YgAWXqDfsThe2fzP-nPDFWdLD2pH7FzU_i8b1C0XjnTTp6flY6GDeiBScEib--AUREF91u29hIxczkWZGJES4uifh4KbpfDxChjYxbrDeFLrn8YU_QU1tRZzqLGxQGrm4LmktMzLt6iFrVAOzO1diwaj8KGhEVr9syi4GB_JwgoOBntTw_CURAM2AuLwtEOhQ5H43A0mU6m43AzYL8b2b2d6d50Pzwcj8aH04O9MAw2fwATHmUb)](https://mermaid.live/edit#pako:eNqNU21P2zAQ_isnf9q0FggNpY20SdDyMmmIl8KQRiZk4kti1bEjx1lhpf9957xANWnS8iX22c_z3D13XrPECGQRyywvc7idxzbWQN9dqQwXDzGbWeTOWKibQAW_pEADK-lyCCaf4EjUykGqeBaznzAcfoH58eMpbQl6ZSqXWVxcfwNZPfLm5mdwtka62wv1_w7WUJwiiusa7cs6Zn4Nu7CoswwrJ42uoDmK2aaHvl334NeYzTHlXmvBU4QLqi9mr3DynKhaIKV1f35yc7KdUcpVhfDhXAqBGlJrCijrJyUTSIm5-riV7d9SdxVayHnVeIGaPykUXu2r7tW6lQCpW7qOrWe85y7J59Ji4h46PlMiVbm78ie7kRRg0oa_sb63eZZjsryyJpUK1x2w28ICnZM6q7Y82r7e5V7lZtV68JgY7VC_d-cVjpQyK0qozQyuFH954snyvfH33GrSgGMusu1-_q9QazoN1llNffWSZ9xho1iVpAYX3NUWYdYhKJwoLgsqk1rK1ZakB3ZSM-raEi41DtsVoVNpC09_a7JM4aX2LpfCQ_os_YgAWXqDfsThe2fzP-nPDFWdLD2pH7FzU_i8b1C0XjnTTp6flY6GDeiBScEib--AUREF91u29hIxczkWZGJES4uifh4KbpfDxChjYxbrDeFLrn8YU_QU1tRZzqLGxQGrm4LmktMzLt6iFrVAOzO1diwaj8KGhEVr9syi4GB_JwgoOBntTw_CURAM2AuLwtEOhQ5H43A0mU6m43AzYL8b2b2d6d50Pzwcj8aH04O9MAw2fwATHmUb)

<details>
<summary><b>View Mermaid Source</b></summary>

```mermaid
graph TD
    Upload["Creator uploads video with 18+ Adult flag"] --> DB_Flag["PostgreSQL is_adult = true"]
    
    DB_Flag --> FeedQuery{"Feed / Suggestions Query"}
    FeedQuery -->|"Default Safe Mode"| Exclude["WHERE is_adult = false (Hidden from public feeds)"]
    FeedQuery -->|"User has 18+ enabled"| Include["Included in feeds"]

    WatchDirect["User opens /watch/:id of 18+ video"] --> CheckProfile{"User Profile Settings"}
    CheckProfile -->|"show_adult_content = true"| Allow["Direct Playback with 18+ Warning Badge"]
    CheckProfile -->|"show_adult_content = false or Guest"| Gate["Display Mature Content Disclaimer Modal"]
    Gate -->|"Click One-Click Confirm"| ToggleOn["Update Profile Mode -> Reload Video"]
    Gate -->|"Click Go Back"| SafeHome["Redirect to Safe Feed"]
```
</details>

---

## ⚡ Key Features

- **Blazing Fast Playback**: Clean video player with responsive aspect-ratio, Cinema/Theater mode toggle (`T`), interactive tag navigation pills, and auto-play recommendations.
- **Direct Cloud Upload**: Stream directly from the browser to cloud storage with live transfer speed metrics, byte progression, drag-and-drop validation, and upload cancellation (`xhr.abort()`).
- **Creator Studio & In-Place Metadata Editor**: Manage published videos, toggle visibility (`public`, `unlisted`, `private`), update tags, categories, adult status, and re-index OpenSearch in real time.
- **Channel Analytics**: Interactive trend graphs displaying views, likes, comments, and subscriber growth over time.
- **Full-Text & Autocomplete Search**: Instant search suggestions powered by OpenSearch edge-ngrams, with transparent fallback to PostgreSQL full-text `tsvector` with GIN indexing.
- **Event-Driven Micro-Batching**: Asynchronous Kafka workers buffer video views to protect PostgreSQL from write contention during high traffic.
- **Server-Sent Events (SSE)**: Instant notifications for new subscriber videos, likes, comments, and replies without WebSocket overhead.
- **Adult Content Protection**: Multi-layer mature content isolation with explicit profile gating and UI warnings.
- **Perpetual Storage Auto-Refresh**: Background worker checks videos reaching Streamtape's 28-day idle expiration threshold and re-clones them automatically.
- **Admin Management Console**: System statistics, platform-wide video moderation, and user management.

---

## 🛠️ Tech Stack

| Layer | Technology | Description |
|---|---|---|
| **Frontend** | SvelteKit (Svelte 5 Runes) | Reactive component architecture with TypeScript |
| **Styling** | Tailwind CSS v4 + Bits UI | Modern dark/light glassmorphic UI design system |
| **Icons** | Lucide Svelte | Sleek, consistent iconography |
| **Backend API** | Go (Fiber v2) | High-performance compiled REST engine |
| **ORM / Database** | PostgreSQL 16 + GORM | Relational database with GIN full-text indexes |
| **Cache & Sessions** | Valkey 8 / Redis | Sub-millisecond key-value caching and feed invalidation |
| **Event Broker** | Apache Kafka 3.8 (KRaft mode) | Distributed event streaming without ZooKeeper |
| **Search Engine** | OpenSearch 2.18 | BM25 relevance search and edge-ngram autocomplete |
| **Cloud Storage** | Aura API / Streamtape | Video hosting, encoding, and CDN streaming |

---

## 🚀 Zero-Config Docker Setup (Recommended)

Run the complete Aurahub stack locally with a single command:

```bash
docker compose up --build
```

*(To run in the background: `docker compose up -d --build`)*

### ✨ What Happens Automatically
1. **PostgreSQL 16**: Starts on port `5432`. Missing tables, custom ENUMs (`video_visibility`, `streamtape_status`, `notification_type`, `interaction_type`), generated `tsvector` columns, and GIN indexes are initialized idempotently.
2. **Valkey 8**: In-memory Redis-compatible engine starts on port `6379`.
3. **Apache Kafka (KRaft)**: Starts on ports `9092` and `29092`. The `kafka-init` container automatically provisions topics:
   - `aurahub.video.lifecycle` (3 partitions)
   - `aurahub.batch-upload-jobs` (3 partitions)
   - `aurahub.video.views` (3 partitions)
   - `aurahub.notifications` (3 partitions)
4. **OpenSearch 2.18**: Starts on port `9200`. The backend automatically creates the `aurahub_videos` index with custom edge-ngram tokenizers.
5. **Backend (Go Fiber)**: Compiles into an Alpine container on port `8080`, listening for requests and starting Kafka workers.
6. **Frontend (SvelteKit)**: Runs on port `5173`, automatically proxying `/api` requests to the backend.

Open your browser to:
👉 **[http://localhost:5173](http://localhost:5173)**

---

### 🔌 Service Ports & Credentials

| Service | Host Port | Internal Port | Default Credentials | Description |
|---|---|---|---|---|
| **Frontend Web App** | `5173` | `5173` | - | [http://localhost:5173](http://localhost:5173) |
| **Backend API** | `8080` | `8080` | - | [http://localhost:8080](http://localhost:8080) |
| **Health Check** | `8080` | `8080` | - | [http://localhost:8080/api/health](http://localhost:8080/api/health) |
| **PostgreSQL** | `5432` | `5432` | `aurahub` / `aurahub_secret` / db: `aurahub` | Relational database |
| **Valkey (Redis)** | `6379` | `6379` | No password | Key-value cache |
| **Kafka Broker** | `29092` | `9092` | PLAINTEXT | Event broker |
| **OpenSearch** | `9200` | `9200` | Security disabled in dev | Full-text search engine |

---

### 🛑 Stopping & Resetting Volumes

To stop all containers:
```bash
docker compose down
```

To stop all containers and wipe local database/cache volumes for a clean reset:
```bash
docker compose down -v
```

---

## 💻 Local / Hybrid Development Setup

If you prefer running the Go backend and SvelteKit frontend natively on your host machine (e.g. for fast hot reload with `air` and Vite):

### Prerequisites
- **Go**: 1.22+ installed
- **Node.js**: 18+ and `npm` installed
- **Docker**: For running PostgreSQL, Valkey, Kafka, and OpenSearch (or use cloud instances)

---

### 1. Backend Setup

1. Copy the example environment file:
   ```bash
   cp backend/.env.example backend/.env
   ```
2. Configure `backend/.env` with your database and service endpoints (see [Environment Configuration](#-environment-configuration-env)).
3. Start the supporting services via Docker:
   ```bash
   docker compose up -d postgres valkey kafka kafka-init opensearch
   ```
4. Run the Go server:
   ```bash
   cd backend
   go run main.go
   ```
   *(Alternatively, use `air` for automatic hot-reloading on code save: `air`)*

---

### 2. Frontend Setup

1. Install frontend dependencies:
   ```bash
   cd frontend
   npm install
   ```
2. Start the development server:
   ```bash
   npm run dev
   ```
3. Visit **[http://localhost:5173](http://localhost:5173)**. Requests to `/api/*` are automatically forwarded to `http://localhost:8080`.

---

### 3. OpenSearch Indexing & Backfill

If you already have existing videos in PostgreSQL and want to index or re-index them into OpenSearch:

```bash
cd backend
go run ./cmd/index-opensearch
```

This script connects to PostgreSQL, ensures the OpenSearch index schema exists with edge-ngram tokenizers, and bulk-indexes all public videos with proper field boosts.

---

## ⚙️ Environment Configuration (.env)

Configure these variables in `backend/.env`:

| Variable | Default / Example | Required | Description |
|---|---|---|---|
| `PORT` | `8080` | No | HTTP port for the Go Fiber server |
| `DATABASE_URL` | `postgres://aurahub:aurahub_secret@localhost:5432/aurahub?sslmode=disable` | **Yes** | PostgreSQL connection string |
| `VALKEY_URL` | `redis://localhost:6379/0` | **Yes** | Valkey / Redis connection URL |
| `JWT_SECRET` | `replace_with_a_long_random_secret` | **Yes** | Secret key for signing authentication tokens |
| `FRONTEND_ORIGINS` | `http://localhost:5173` | No | Allowed CORS origins (comma-separated) |
| `UPLOAD_FOLDER_ID` | `your_folder_id` | No | Default destination folder on Streamtape/Aura |
| `FREEIMAGE_API_KEY` | `your_api_key` | No | API key for external thumbnail hosting |
| `KAFKA_BROKERS` | `localhost:29092` (or cloud host) | No | Comma-separated list of Kafka broker endpoints |
| `KAFKA_USER` | `avnadmin` | No | SASL username (for cloud brokers like Aiven) |
| `KAFKA_PASSWORD` | `your_password` | No | SASL password |
| `KAFKA_AUTH_METHOD` | `scram-sha-512` or `plain` | No | SASL mechanism (`scram-sha-256`, `scram-sha-512`, `plain`) |
| `KAFKA_INSECURE_SKIP_VERIFY` | `false` | No | Skip TLS certificate verification |
| `KAFKA_CA_CERT_PATH` | `./certs/ca.pem` | No | Path to CA certificate for mTLS authentication |
| `KAFKA_CLIENT_CERT_PATH` | `./certs/service.cert` | No | Path to client certificate for mTLS authentication |
| `KAFKA_CLIENT_KEY_PATH` | `./certs/service.key` | No | Path to client private key for mTLS authentication |
| `OPENSEARCH_URL` | `http://localhost:9200` | No | OpenSearch cluster endpoint |
| `OPENSEARCH_USER` | `admin` | No | OpenSearch HTTP Basic auth user |
| `OPENSEARCH_PASSWORD` | `admin` | No | OpenSearch HTTP Basic auth password |
| `OPENSEARCH_INSECURE_SKIP_VERIFY` | `false` | No | Skip OpenSearch TLS certificate verification |

---

## 📂 Project Directory Structure

```text
Aurahub/
├── docker-compose.yml              # All-in-one stack orchestration (PostgreSQL, Valkey, Kafka, OpenSearch, Go, SvelteKit)
├── README.md                       # Comprehensive platform documentation and flow diagrams
│
├── backend/                        # Go (Fiber) REST API & Background Workers
│   ├── api/                        # HTTP route handlers (Auth, Videos, Uploads, Creator, Playlists, Admin, SSE)
│   ├── cmd/
│   │   ├── index-opensearch/       # CLI utility to backfill/re-index videos into OpenSearch
│   │   ├── initdb/                 # Schema initialization CLI tool
│   │   └── migrate-mongo/          # Data migration utility from legacy MongoDB
│   ├── config/                     # Environment and configuration loading
│   ├── db/                         # Database connection, GORM repository, and schema.sql
│   ├── internal/
│   │   ├── cache/                  # Valkey / Redis caching helpers
│   │   ├── kafka/                  # Kafka producer, consumer, TLS/SASL auth, and events
│   │   │   └── workers/            # Background workers (ViewsFlusher, OpenSearchIndexer, AutoRefresh)
│   │   └── opensearch/             # OpenSearch client, query builders, and index schema
│   ├── middleware/                 # JWT Auth, CORS, Logger, and Recovery middlewares
│   ├── streamtape/                 # Streamtape / Aura API client
│   ├── main.go                     # API entry point & graceful shutdown lifecycle
│   └── Dockerfile                  # Lightweight multi-stage Alpine Go container
│
└── frontend/                       # SvelteKit Web Application
    ├── src/
    │   ├── lib/
    │   │   ├── components/         # Reusable Svelte components (VideoPlayer, Navbar, EditVideoModal, etc.)
    │   │   ├── stores/             # Global rune-based state (auth, content mode, notifications)
    │   │   └── utils.ts            # Formatting and helper utilities
    │   └── routes/                 # File-based routing
    │       ├── +page.svelte        # Home video discovery feed
    │       ├── watch/[id]/         # Video watch page with Theater mode & recommendations
    │       ├── upload/             # Direct video uploader with speed stats & tag chips
    │       ├── dashboard/          # Creator Studio with metadata editor & video manager
    │       ├── analytics/          # Interactive channel performance graphs
    │       ├── admin/              # Global platform administration console
    │       └── search/             # Multi-faceted search results page
    ├── static/                     # Static media and favicons
    ├── vite.config.ts              # Vite bundler & backend API proxy configuration
    └── Dockerfile                  # Node.js container for frontend deployment
```

---

## ❓ Troubleshooting & FAQs

### 1. `Kafka: context deadline exceeded` during video views
- **Cause**: Outbound network latency or cold TLS handshakes when connecting to cloud brokers (e.g. Aiven) from a local environment.
- **Resolution**: Aurahub includes connection pooling (`IdleTimeout: 30s`) and an extended asynchronous goroutine timeout (`15s`). The background worker automatically aggregates and flushes views without dropping events.

### 2. OpenSearch memory limit error in Docker
- **Cause**: OpenSearch requires sufficient JVM heap and memory lock limits.
- **Resolution**: `docker-compose.yml` pre-configures `OPENSEARCH_JAVA_OPTS="-Xms512m -Xmx512m"`. On Linux hosts, ensure `vm.max_map_count` is set:
  ```bash
  sudo sysctl -w vm.max_map_count=262144
  ```

### 3. Missing tables or database schema errors on boot
- **Cause**: Reusing an old PostgreSQL volume with mismatched schema.
- **Resolution**: Run `docker compose down -v` to reset data volumes, or run `go run ./backend/cmd/initdb` to verify the schema.

---

## 📄 License
This project is licensed under the [MIT License](LICENSE).
