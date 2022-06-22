ARG CENTOS_VERSION=centos-7.6-builder:1.0.67
# ARG CENTOS_VERSION=centos-8-builder:2.0.13
FROM docker-registry.activestate.build/activestate/${CENTOS_VERSION} as offline-installer-builder

USER root

# Install required basic commands
RUN yum install -y git go curl
RUN yum install -y gtk3-devel libappindicator-gtk3 webkitgtk4
ENV GOPATH=/go
ENV PATH=$PATH:/go/bin

# Install gozip which we use to add files to our installer
RUN go install github.com/ActiveState/gozip/cmd/gozip@latest
RUN chown -R as-build:as-build /go/pkg

# Install State Tool
ARG ACTIVESTATE_API_KEY
ARG ACTIVESTATE_API_HOST

RUN echo "API KEY: "  $ACTIVESTATE_API_KEY
RUN echo "API HOST: " $ACTIVESTATE_API_HOST

# Fetch ActivePython using the state tool
# RUN curl -s https://platform.activestate.com/dl/cli/install.sh | sh -s /state-tool && \
#     /state-tool/bin/state auth --non-interactive --token "$ACTIVESTATE_API_KEY"
RUN curl -s https://platform.activestate.com/dl/cli/install.sh | sh -s /state-tool
    
ENV PATH=$PATH:/state-tool/bin

# Get the state tool code, so we can build the offline installer
WORKDIR /var/builds

USER as-build

RUN mkdir -p /home/as-build/.cache/go-build:

RUN git clone https://github.com/ActiveState/cli.git

COPY runCompile.sh /usr/local/bin/runCompile

