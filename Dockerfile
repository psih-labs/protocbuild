FROM golang:1.15-alpine as builder
RUN apk add --no-cache ca-certificates git

WORKDIR /go/src/protocbuild

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o appbinary

FROM alpine as release

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/protocbuild/appbinary /appbinary
VOLUME /
ENTRYPOINT ["/appbinary"]
