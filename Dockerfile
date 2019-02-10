FROM golang:alpine AS builder

# install git
RUN apk update && apk add --no-cache git

WORKDIR $GOPATH/src/github.com/nadilas/godaddy-oc-updater
COPY . .

# fetch dependencies
RUN go get -d -v

# build the binary
RUN go build -o /opt/ip-updater

# build production image
FROM scratch

# copy executable
COPY --from=builder /opt/ip-updater /opt/ip-updater

ENTRYPOINT ["/opt/ip-updater"]