FROM golang:1.12.1

WORKDIR /app

RUN apt-get update
RUN apt-get -y install python3 python3-pip

RUN go get github.com/gobuffalo/packr/packr
RUN go get github.com/jstemmer/go-junit-report

RUN curl -L https://codeclimate.com/downloads/test-reporter/test-reporter-0.6.3-linux-amd64 > /usr/bin/cc-test-reporter
RUN chmod +x /usr/bin/cc-test-reporter

ENV GOFLAGS=-mod=vendor