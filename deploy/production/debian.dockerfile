FROM --platform=$TARGETPLATFORM golang:latest AS builder

ARG TARGETPLATFORM
ARG BUILDPLATFORM

# Create appuser.
ENV USER=appuser
ENV UID=10001

# See https://stackoverflow.com/a/55757473/12429735RUN 
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

# RUN apk update && apk upgrade && apk add --no-cache ca-certificates
RUN apt-get update && apt-get install -y ca-certificates && apt-get clean
RUN update-ca-certificates

RUN cp /usr/local/go/lib/time/zoneinfo.zip /var/zoneinfo.zip

###############################################################################
FROM --platform=$TARGETPLATFORM debian:stable-slim

ARG TARGETPLATFORM
ARG BUILDPLATFORM

LABEL maintainer="GeniusRabbit (Dmitry Ponomarev github.com/demdxx)"
LABEL service.name=apfs
LABEL service.veight=1

ENV LOG_LEVEL=info \
    STORAGE_METADB_CONNECT=sqlite3:///data/apfs.db?cache=shared \
    STORAGE_STATE_CONNECT=memory \
    STORAGE_AUTOMIGRATE=true

# Import the user and group files from the builder.
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /var/zoneinfo.zip /usr/local/go/lib/time/
COPY .build/${TARGETPLATFORM}/apfs /

# Use an unprivileged user.
USER appuser:appuser

ENTRYPOINT ["/apfs", "server", "--processing=1"]