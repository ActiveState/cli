FROM golang:1.16.13-buster
RUN apt-get update && apt-get -y install go-dep upx-ucl
ADD https://platform.activestate.com/dl/cli/install.sh /tmp/install.sh
RUN TERM=xterm sh /tmp/install.sh -n
WORKDIR /go/src/github.com/ActiveState/cli
CMD state auth --token $APITOKEN && state run build && state run test
