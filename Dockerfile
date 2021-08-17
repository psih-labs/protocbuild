FROM golang:1.15-alpine as builder
RUN apk add --no-cache ca-certificates git

WORKDIR /go/src/protocbuild

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o appbinary

FROM debian:buster-slim as release

COPY --from=thethingsindustries/protoc /usr/bin/ /usr/local/bin/
COPY --from=thethingsindustries/protoc /usr/include/ /usr/include/
COPY --from=namely/protoc-all /usr/local/bin/ /usr/local/bin/
COPY --from=namely/protoc-all /usr/local/include/ /usr/local/include/
COPY --from=namely/protoc-all /usr/local/lib/ /usr/local/lib/
COPY --from=namely/protoc-all /usr/local/share/ /usr/local/share/
COPY --from=namely/protoc-all /opt/include/google /usr/local/include/google

RUN apt-get update && apt-get install ca-certificates -y --no-install-recommends 
COPY --from=builder /go/src/protocbuild/appbinary /usr/local/bin/appbinary
COPY --from=builder /protos/* /usr/local/include/*

VOLUME /workspace
ENTRYPOINT ["appbinary"]
