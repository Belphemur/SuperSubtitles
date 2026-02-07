# Runtime stage
FROM alpine:latest

ENV ENV="production"

WORKDIR /app

ARG USER=default
ARG TARGETPLATFORM
ENV HOME=/app/

RUN apk add --no-cache ca-certificates

# add new user
RUN adduser -D $USER

# Copy pre-built binary from the platform-specific directory
COPY ${TARGETPLATFORM}/super-subtitles /app/super-subtitles

# Copy default configuration
COPY config/config.yaml /app/config/config.yaml

RUN chown -R $USER:$USER -R /app/

USER $USER:$USER

# Expose port for the proxy server
EXPOSE 8080

# Volume for persistent data
VOLUME ["/app/data"]

CMD ["/app/super-subtitles"]
