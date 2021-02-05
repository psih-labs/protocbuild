FROM golang:1.15-alpine as builder
RUN apk add --no-cache ca-certificates git

WORKDIR /go/src/protocbuild

COPY go.mod go.sum ./
RUN go mod download

COPY . ./
RUN go build -o appbinary

FROM alpine as release
RUN apk add --no-cache ca-certificates git openssh docker-cli rsync
COPY --from=builder /go/src/protocbuild/appbinary /appbinary
COPY --from=builder /go/src/protocbuild/run.sh /run.sh
COPY --from=builder /go/src/protocbuild/setupgit.sh /setupgit.sh

ENV SSH_KEY=""
ENV COMMIT_USER=""
ENV COMMIT_EMAIL=""

RUN chmod +x run.sh
RUN chmod +x setupgit.sh

VOLUME /workspace
ENTRYPOINT ["/appbinary"]
