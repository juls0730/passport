FROM golang:1.25 AS builder

# build dependencies
RUN apt update && apt install -y upx unzip

RUN curl -fsSL https://bun.com/install | BUN_INSTALL=/usr bash

ARG TARGETARCH
RUN set -eux; \
    echo "Building for architecture: ${TARGETARCH}"; \
    case "${TARGETARCH}" in \
        "amd64") \
            arch_suffix='x64' ;; \
        "arm64") \
            arch_suffix='arm64' ;; \
        *) \
            echo "Unsupported architecture: ${TARGETARCH}" && exit 1 ;; \
    esac; \
    curl -sLO "https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.13/tailwindcss-linux-${arch_suffix}"; \
    mv "tailwindcss-linux-${arch_suffix}" /usr/local/bin/tailwindcss; \
    chmod +x /usr/local/bin/tailwindcss;
    

RUN go install github.com/juls0730/zqdgr@latest

WORKDIR /app

ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH}

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN bun install

RUN zqdgr build
RUN upx passport

# ---- Runtime Stage ----
FROM gcr.io/distroless/static-debian12 AS runner

WORKDIR /data
COPY --from=builder /app/passport /usr/local/bin/passport
EXPOSE 3000

CMD ["/usr/local/bin/passport"]