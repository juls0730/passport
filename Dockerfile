FROM golang:1.25 AS builder

# build dependencies
RUN apt update && apt install -y upx unzip

RUN curl -fsSL https://bun.com/install | BUN_INSTALL=/usr bash
    

RUN go install github.com/juls0730/zqdgr@v0.0.6-1

WORKDIR /app

ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN bun install

RUN zqdgr build

# ---- Runtime Stage ----
FROM gcr.io/distroless/static-debian12 AS runner

WORKDIR /data
COPY --from=builder /app/passport /usr/local/bin/passport
EXPOSE 3000

CMD ["/usr/local/bin/passport"]