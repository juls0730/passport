FROM golang:1.25 AS builder

# build dependencies
RUN apt update && apt install -y upx

RUN curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/download/v4.1.13/tailwindcss-linux-x64
RUN chmod +x tailwindcss-linux-x64
RUN mv tailwindcss-linux-x64 /usr/local/bin/tailwindcss

RUN go install github.com/juls0730/zqdgr@latest

WORKDIR /app

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

COPY go.mod go.sum ./
RUN go mod download
COPY . .


RUN zqdgr build
RUN upx passport

# ---- Runtime Stage ----
FROM gcr.io/distroless/static-debian12 AS runner

WORKDIR /data
COPY --from=builder /app/passport /usr/local/bin/passport
EXPOSE 3000

CMD ["/usr/local/bin/passport"]