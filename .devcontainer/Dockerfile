FROM mcr.microsoft.com/devcontainers/go:latest

# Install additional tools
RUN apt-get update && apt-get install -y \
    git \
    curl \
    vim \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory
WORKDIR /workspace

# Copy go.mod for dependency caching
COPY go.mod .

# Download dependencies (will create go.sum if needed)
RUN go mod download

# Ensure the workspace is owned by vscode user
RUN chown -R vscode:vscode /workspace

# Switch to vscode user (the default user in devcontainer)
USER vscode

# Keep the container running
CMD ["sleep", "infinity"]
