# Deployment Guide

## Required environment variables

Create `.env` in project root:

```env
DB_ROOT_PASSWORD=change-me
DB_PASSWORD=change-me
JWT_SECRET=change-me-long-random-string
DB_USER=witchhunt
DB_NAME=witchhunt
HOST_PORT=80
```

## Production

```bash
docker compose -f docker-compose.yml -f docker-compose.prod.yml --profile prod up -d --build
```

## Development

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile dev up -d --build
```

## Verify

```bash
docker compose ps
docker compose logs -f --tail=200
docker stats
```
