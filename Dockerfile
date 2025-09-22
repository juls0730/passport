FROM golang:1.25 AS builder

WORKDIR /app

ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN apt-get update && apt-get install -y gcc libc6-dev sqlite3 ca-certificates

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -ldflags="-w -s" -o passport 

# ---- Runtime Stage ----
FROM gcr.io/distroless/cc-debian12

WORKDIR /data
COPY --from=builder /app/passport /usr/local/bin/passport
EXPOSE 3000

CMD ["/usr/local/bin/passport"]