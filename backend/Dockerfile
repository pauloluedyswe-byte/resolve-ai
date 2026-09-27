FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.22
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/api /app/api
RUN mkdir -p /app/uploads && chown app /app/uploads
USER app
ENV PORT=8080 UPLOAD_DIR=/app/uploads
EXPOSE 8080
ENTRYPOINT ["/app/api"]
