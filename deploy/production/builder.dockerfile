FROM golang:latest

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y git && \
    apt-get install -y libva-dev && \
    apt-get install -y libc6 && \
    apt-get clean

ENV GOPATH=/project/.tmp/