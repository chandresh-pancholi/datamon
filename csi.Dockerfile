FROM golang:alpine as base

ARG github_user
ARG github_token

ENV GITHUB_USER ${github_user}
ENV GITHUB_TOKEN ${github_token}

ADD hack/create-netrc.sh /usr/bin/create-netrc

RUN mkdir -p /stage/data /stage/etc/ssl/certs &&\
  create-netrc &&\
  apk add --no-cache musl-dev gcc ca-certificates mailcap upx tzdata zip git &&\
  update-ca-certificates &&\
  cp /etc/ssl/certs/ca-certificates.crt /stage/etc/ssl/certs/ca-certificates.crt &&\
  cp /etc/mime.types /stage/etc/mime.types

# https://golang.org/src/time/zoneinfo.go Copy the zoneinfo installed by musl-dev
WORKDIR /usr/share/zoneinfo
RUN zip -r -0 /stage/zoneinfo.zip .

ADD . /datamon
WORKDIR /datamon

RUN go build -o /stage/usr/bin/datamon-csi --ldflags '-s -w -linkmode external -extldflags "-static"' ./cmd/csi
RUN upx /stage/usr/bin/datamon-csi
RUN md5sum /stage/usr/bin/datamon-csi

# Build the dist image
FROM ubuntu:latest
RUN apt-get update && apt-get install -y --no-install-recommends fuse &&\
  apt-get clean && rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
COPY --from=base /stage /
ENV ZONEINFO /zoneinfo.zip
ENTRYPOINT [ "datamon-csi" ]
CMD ["--help"]

