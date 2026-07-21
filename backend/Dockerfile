FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/job-hunter-agent ./cmd/server

FROM alpine:3.22

WORKDIR /app
RUN adduser -D -H appuser && mkdir -p /data && chown -R appuser:appuser /data
COPY --from=build /out/job-hunter-agent /app/job-hunter-agent

ENV APP_ADDR=:8080
ENV APP_DB_PATH=/data/job-hunter-agent.db
EXPOSE 8080

USER appuser
CMD ["/app/job-hunter-agent"]
