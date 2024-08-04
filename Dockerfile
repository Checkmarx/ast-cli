# Use Debian as the base image
FROM debian:latest

# Switch to root user to update curl
USER root

# Update the package list and upgrade curl to the fixed version
RUN apt-get update && \
    apt-get install -y curl=8.9.1-1 && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Switch back to non-root user
USER nonroot

# Copy the application binary
COPY cx /app/bin/cx

# Set the entrypoint to the application binary
ENTRYPOINT ["/app/bin/cx"]
