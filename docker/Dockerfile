# STEP 1 build executable binary

FROM golang:1.26.2-alpine3.23 AS builder
RUN apk add --no-cache git

# Create appuser on builder image
RUN adduser -D -g '' appuser

WORKDIR /tmp/app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY cmd ./cmd

RUN go env -w CGO_ENABLED=0
RUN go build -o /tmp/eks-auto-pod-id-assoc github.com/udhos/eks-auto-pod-id-assoc/cmd/eks-auto-pod-id-assoc
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -o /tmp/eks-auto-pod-id-assoc ./cmd/eks-auto-pod-id-assoc
#
# STEP 2 build a small image from alpine
#
# curl: https://security.alpinelinux.org/srcpkg/curl
#
FROM alpine:3.23.3
COPY --from=builder /tmp/eks-auto-pod-id-assoc /bin/eks-auto-pod-id-assoc
#RUN apk add curl=8.1.2-r0 libcrypto3=3.1.0-r4 libssl3=3.1.0-r4
RUN apk update && \
    apk add curl && \
    apk upgrade libssl3 libcrypto3 busybox busybox-binsh ssl_client libcurl nghttp2-libs zlib && \
    adduser -D -g '' user && \
    rm -rf /var/cache/apk/*
USER user
ENTRYPOINT ["/bin/eks-auto-pod-id-assoc"]
