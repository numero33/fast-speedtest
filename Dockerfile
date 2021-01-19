FROM golang:alpine as builder

RUN mkdir -p /go/src/github.com/numero33/fast-speedtest
WORKDIR /go/src/github.com/numero33/fast-speedtest
COPY main.go .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-s -w' -o build/fast-speedtest

FROM alpine:latest

RUN apk --update --no-cache add ca-certificates && rm -rf /var/cache/apk/*
COPY --from=builder /go/src/github.com/numero33/fast-speedtest/build/fast-speedtest /fast-speedtest
RUN chmod +x /fast-speedtest
EXPOSE 80
ENTRYPOINT ["/fast-speedtest"]