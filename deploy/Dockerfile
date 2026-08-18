FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/telemetry-alert ./cmd/telemetry-alert

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/telemetry-alert /app/telemetry-alert
COPY configs/config.yaml /app/configs/config.yaml
RUN mkdir -p /app/data && chown -R app:app /app
USER app
EXPOSE 8080
ENTRYPOINT ["/app/telemetry-alert", "-config", "/app/configs/config.yaml"]
