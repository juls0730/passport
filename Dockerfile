FROM golang:1.25 AS builder

WORKDIR /app

ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64
RUN apt-get update && apt-get install -y gcc libc6-dev sqlite3 ca-certificates

COPY go.mod go.sum ./
RUN go mod download
COPY . .

# tailwindcss needed for go generate
RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.13/tailwindcss-linux-x64
RUN chmod +x tailwindcss-linux-x64
RUN mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss

RUN go generate
RUN go build -ldflags="-w -s" -o passport 

# ---- Runtime Stage ----
FROM gcr.io/distroless/cc-debian12

WORKDIR /data
COPY --from=builder /app/passport /usr/local/bin/passport
EXPOSE 3000

CMD ["/usr/local/bin/passport"]