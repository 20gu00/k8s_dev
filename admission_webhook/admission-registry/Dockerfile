# Build the webhook binary
#golang-alpine
FROM golang:1.18.5 as builder

ENV CGO_ENABLED 0
#ENV CGO_ENABLED=0
ENV GOOS linux
ENV GOARCH amd64
ENV GO111MODULE on
#ENV GOPROXY="https://goproxy.cn"
ENV GOPROXY https://goproxy.cn,direct
#RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories
#RUN apk update --no-cache && apk add --no-cache upx

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
#COPY main.go main.go
#COPY common/ common/
COPY . .

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
#-ldflags="-s -w"镜像更小
RUN go build -ldflags="-s -w" -o /workspace/admission-registry main.go  &&
    go build -ldflags="-s -w" -o /workspace/tls-manager tls/main.go
    #&& \
    #upx admission-registry

#FROM alpine:3.9.2
FROM scratch as admission
COPY --from=builder /workspace/admission-registry .
ENTRYPOINT ["/admission-registry"]

FROM scratch as tls
COPY --from=builder /workspace/tls .
ENTRYPOINT ["/tls"]
# --target制定即可生成不同的镜像