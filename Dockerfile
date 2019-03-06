# based on https://github.com/gliderlabs/registrator/blob/master/Dockerfile
FROM alpine:3.7 AS builder
COPY . /go/src/github.com/nadilas/godaddy-oc-updater
RUN apk --no-cache add -t build-deps build-base go git curl \
	&& apk --no-cache add ca-certificates \
	&& export GOPATH=/go && mkdir -p /go/bin && export PATH=$PATH:/go/bin \
	&& curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh \
	&& cd /go/src/github.com/nadilas/godaddy-oc-updater \
	&& export GOPATH=/go \
	&& git config --global http.https://gopkg.in.followRedirects true \
	&& dep ensure \
	&& go build -ldflags "-X main.Version=$(cat VERSION)" -o /bin/ip-updater \
	&& rm -rf /go \
	&& apk del --purge build-deps

FROM alpine:3.7
COPY --from=builder /bin/ip-updater /bin/ip-updater
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ENTRYPOINT ["/bin/ip-updater"]