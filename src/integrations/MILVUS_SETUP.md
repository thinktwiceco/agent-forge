# Milvus Local Setup

This directory contains Docker Compose configuration to run Milvus locally for development and testing.

## Prerequisites

- Docker
- Docker Compose

## Quick Start

1. Start Milvus and its dependencies:
```bash
docker-compose -f docker-compose.milvus.yml up -d
```

2. Verify Milvus is running:
```bash
docker-compose -f docker-compose.milvus.yml ps
```

3. Check Milvus health:
```bash
curl http://localhost:9091/healthz
```

## Services

- **Milvus**: Vector database server running on port `19530`
- **etcd**: Metadata storage (internal)
- **MinIO**: Object storage for Milvus (console on port `9001`)

## Configuration

The default configuration matches the settings in `cmd/chat/main.go`:
- Host: `localhost`
- Port: `19530`
- Collection: `agent_knowledge`
- Vector Dimension: `1536` (for text-embedding-3-small)

## Stopping Services

```bash
docker-compose -f docker-compose.milvus.yml down
```

To remove all data volumes:
```bash
docker-compose -f docker-compose.milvus.yml down -v
```

## Accessing MinIO Console

- URL: http://localhost:9001
- Username: `minioadmin`
- Password: `minioadmin`

## Troubleshooting

1. **Port already in use**: Ensure ports 19530, 9000, 9001, and 2379 are not in use
2. **Connection refused**: Wait a few seconds for all services to start up
3. **Check logs**: `docker-compose -f docker-compose.milvus.yml logs milvus`

