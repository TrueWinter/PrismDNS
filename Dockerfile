FROM golang:1.26.3-alpine3.22 AS build

RUN apk add bash git

ARG VERSION
ENV VERSION=$VERSION

WORKDIR /app

COPY . .
ENV CGO_ENABLED=0
RUN chmod +x build.sh && /bin/bash ./build.sh

#FROM alpine:3.22.4 AS server
FROM scratch AS server

COPY --from=build /app/dist/PrismDNS-linux-amd64 /app/server
ENTRYPOINT ["/app/server"]