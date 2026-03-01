# syntax=docker/dockerfile:1
FROM node:20-alpine AS builder

WORKDIR /build
COPY package.json package-lock.json* ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci --maxsockets=2
COPY . .
RUN NODE_OPTIONS="--max-old-space-size=384" npm run build

FROM nginx:alpine
RUN apk add --no-cache wget
COPY --from=builder /build/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
