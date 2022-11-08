FROM golang:1.18-alpine AS builder

ENV GOPROXY https://goproxy.cn,direct
ENV GOOS linux
ENV GO111MODULE auto

WORKDIR /go/src/app

ADD go.mod .
ADD go.sum .
RUN go mod download
COPY . .

RUN go build -ldflags="-s -w" -o /app/clientset clientset.go

FROM alpine

ENV TZ Asia/Shanghai

WORKDIR /app

COPY --from=builder /app/clientset /app/clientset

CMD ["./clientset"]
