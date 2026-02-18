# Download stage - only used to fetch grpc_health_probe
FROM alpine:latest AS downloader

ARG TARGETPLATFORM
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.45

RUN apk add --no-cache wget

# Download grpc_health_probe for health checks with checksum verification
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
    ARCH="amd64"; \
    elif [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
    ARCH="arm64"; \
    else \
    echo "Unsupported platform: $TARGETPLATFORM" && exit 1; \
    fi && \
    BINARY_NAME="grpc_health_probe-linux-${ARCH}" && \
    # Download checksums file \
    wget -qO/tmp/checksums.txt https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/checksums.txt && \
    # Download binary \
    wget -qO/tmp/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/${BINARY_NAME} && \
    # Verify checksum \
    cd /tmp && \
    grep "${BINARY_NAME}" checksums.txt | awk '{print $1}' > expected_checksum.txt && \
    sha256sum grpc_health_probe | awk '{print $1}' > actual_checksum.txt && \
    EXPECTED=$(cat expected_checksum.txt) && \
    ACTUAL=$(cat actual_checksum.txt) && \
    if [ "$EXPECTED" != "$ACTUAL" ]; then \
    echo "Checksum verification failed!" && \
    echo "Expected: $EXPECTED" && \
    echo "Actual: $ACTUAL" && \
    exit 1; \
    fi && \
    echo "Checksum verification passed: $ACTUAL" && \
    chmod +x /tmp/grpc_health_probe

# Runtime stage - minimal final image
FROM alpine:latest

ENV ENV="production"

WORKDIR /app

ARG USER=default
ARG TARGETPLATFORM
ENV HOME=/app/

ENV APP_SERVER_ADDRESS="0.0.0.0"
ENV APP_SERVER_PORT="8080"

RUN apk add --no-cache ca-certificates

# add new user
RUN adduser -D $USER

# Copy grpc_health_probe from download stage
COPY --from=downloader /tmp/grpc_health_probe /bin/grpc_health_probe

# Copy pre-built binary from the platform-specific directory
COPY ${TARGETPLATFORM}/SuperSubtitles /app/super-subtitles

# Copy default configuration
COPY config/config.yaml /app/config/config.yaml

RUN chown -R $USER:$USER -R /app/

USER $USER:$USER

# Expose port for the proxy server
EXPOSE 8080

# Health check using grpc_health_probe
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/bin/grpc_health_probe", "-addr=:8080"]

CMD ["/app/super-subtitles"]
