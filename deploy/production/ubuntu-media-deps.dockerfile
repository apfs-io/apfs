FROM ghcr.io/apfs-io/apfs:ubuntu-imagemagick-deps-latest

ARG TARGETPLATFORM
ARG BUILDPLATFORM

RUN apt-get update \
 && apt-get install -y ffmpeg \
 && apt-get clean
