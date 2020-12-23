# based on https://github.com/gliderlabs/registrator/blob/master/Dockerfile
FROM golang:1.14-alpine3.12 AS builder
ARG version
ENV VERSION=$version
COPY . /opt/godaddy-oc-updater
RUN apk --no-cache add -t curl \
	&& apk --no-cache add ca-certificates \
	&& cd /opt/godaddy-oc-updater \
	&& go build -ldflags "-X main.Version=$(echo $VERSION)" -o /bin/ip-updater

FROM alpine:3.12
COPY --from=builder /bin/ip-updater /bin/ip-updater
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/bin/ip-updater"]