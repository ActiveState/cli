FROM ubuntu:22.04

# Install necessary tools for testing and debugging
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    netcat-openbsd \
    tcpdump \
    strace \
    lsof \
    procps \
    vim \
    less \
    && rm -rf /var/lib/apt/lists/*

# Create working directory
WORKDIR /opt/state-test

# Create the proper payload structure that the installer expects
# The installer expects:
# - state-installer in the root
# - All other binaries in a bin/ subdirectory
# - An install marker file

# Copy state-installer to root
COPY build/state-installer ./

# Create bin directory and copy other binaries there
RUN mkdir -p bin
COPY build/state ./bin/
COPY build/state-exec ./bin/
COPY build/state-svc ./bin/
COPY build/state-mcp ./bin/
COPY build/state-remote-installer ./bin/

# Make binaries executable
RUN chmod +x state-installer
RUN chmod +x bin/state bin/state-exec bin/state-svc bin/state-mcp bin/state-remote-installer

# Create install marker file (required by the installer)
RUN echo '{"channel":"release","version":"test"}' > .state_install_root

# Create a test installation directory
RUN mkdir -p /opt/state-install

# Set environment variables for testing
ENV VERBOSE=true
ENV TERM=xterm-256color

# Create a script to run the installer with config flags
COPY ksa-test/test-installer.sh ./
RUN chmod +x test-installer.sh

# Create a script for network monitoring
COPY ksa-test/monitor-network.sh ./
RUN chmod +x monitor-network.sh

# Default command - run the test installer
CMD ["./test-installer.sh"]
