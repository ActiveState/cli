FROM golang:1.12.9-buster
RUN apt-get update && apt-get -y install go-dep upx-ucl
RUN go get -u github.com/gobuffalo/packr/packr
RUN go get -u github.com/gobuffalo/packr
ADD https://platform.activestate.com/dl/cli/install.sh /tmp/install.sh
RUN TERM=xterm sh /tmp/install.sh -n
WORKDIR /go/src/github.com/ActiveState/cli
CMD state auth --token $APITOKEN && state run build && state run test
