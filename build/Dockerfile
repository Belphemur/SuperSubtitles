# Download stage - only used to fetch grpc_health_probe
FROM alpine:latest AS downloader

ARG TARGETPLATFORM
ARG GRPC_HEALTH_PROBE_VERSION=v0.4.45

RUN apk add --no-cache wget

# Download grpc_health_probe for health checks
RUN if [ "$TARGETPLATFORM" = "linux/amd64" ]; then \
    ARCH="amd64"; \
    elif [ "$TARGETPLATFORM" = "linux/arm64" ]; then \
    ARCH="arm64"; \
    else \
    echo "Unsupported platform: $TARGETPLATFORM" && exit 1; \
    fi && \
    wget -qO/tmp/grpc_health_probe https://github.com/grpc-ecosystem/grpc-health-probe/releases/download/${GRPC_HEALTH_PROBE_VERSION}/grpc_health_probe-linux-${ARCH} && \
    chmod +x /tmp/grpc_health_probe

# Runtime stage - minimal final image
FROM alpine:latest

ENV ENV="production"

WORKDIR /app

ARG USER=default
ARG TARGETPLATFORM
ENV HOME=/app/

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
