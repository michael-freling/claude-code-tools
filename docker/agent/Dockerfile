FROM ubuntu:24.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Node.js LTS (via nodesource)
RUN apt-get update && apt-get install -y ca-certificates curl gnupg \
    && mkdir -p /etc/apt/keyrings \
    && curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg \
    && echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_22.x nodistro main" > /etc/apt/sources.list.d/nodesource.list \
    && apt-get update && apt-get install -y nodejs \
    && rm -rf /var/lib/apt/lists/*

# Python 3
RUN apt-get update && apt-get install -y \
    python3 python3-pip python3-venv \
    && rm -rf /var/lib/apt/lists/*

# Go (copy from official image)
COPY --from=golang:1.25 /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/home/user/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Docker daemon + CLI (for DinD)
RUN apt-get update && apt-get install -y docker.io \
    && rm -rf /var/lib/apt/lists/*

# Common CLIs used by Claude Code
RUN apt-get update && apt-get install -y \
    bash git curl jq make ripgrep \
    tar unzip openssh-client \
    && rm -rf /var/lib/apt/lists/*

# Claude Code (latest)
RUN npm install -g @anthropic-ai/claude-code

# Copy claude-forge binary (built in CI or locally)
COPY claude-forge /usr/local/bin/claude-forge

# Create symlinks for forge-gh and gh
RUN ln -s /usr/local/bin/claude-forge /usr/local/bin/forge-gh \
    && ln -s /usr/local/bin/claude-forge /usr/local/bin/gh

# Create non-root user (entrypoint adjusts UID/GID at startup)
RUN useradd -m -s /bin/bash user

# Entrypoint handles UID/GID mapping and runs as the user
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

WORKDIR /work
ENTRYPOINT ["entrypoint.sh"]
CMD ["claude"]
