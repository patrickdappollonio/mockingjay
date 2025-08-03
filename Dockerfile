FROM gcr.io/distroless/static:nonroot

# Set up directories
WORKDIR /app

# Copy the binary from goreleaser build context
COPY mockingjay /usr/local/bin/mockingjay

# Expose default port
EXPOSE 8080

# Default command
ENTRYPOINT ["/usr/local/bin/mockingjay"]
CMD ["-config=/app/config.yaml", "-port=8080"]
