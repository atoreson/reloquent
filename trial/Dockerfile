# Stage 1: Build web UI
FROM node:20-alpine AS web-builder
COPY web/package.json web/package-lock.json /app/web/
WORKDIR /app/web
RUN npm ci
COPY web/ /app/web/
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache git
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download
COPY . /app/
COPY --from=web-builder /app/web/dist /app/web/dist
RUN CGO_ENABLED=0 go build -o /reloquent .

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=go-builder /reloquent /usr/local/bin/reloquent
COPY trial/config/reloquent-trial.yaml /app/config/reloquent-trial.yaml
EXPOSE 8230
ENTRYPOINT ["reloquent"]
CMD ["serve", "--port", "8230"]
