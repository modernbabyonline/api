# build from vendor directory
FROM golang:1.11.1
WORKDIR /go/src/github.com/modernbabyonline/api/
ADD . .
RUN set -ex && \      
  CGO_ENABLED=0 go build \
        -tags netgo \
        -v -a \
        -ldflags '-extldflags "-static"' \
        -o api

# use 2 step process
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=0 /go/src/github.com/modernbabyonline/api/api .
ENTRYPOINT ["./api"]
EXPOSE 8000
