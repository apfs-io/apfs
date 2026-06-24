FROM golang:1.25

EXPOSE 8080 8081 8082

ARG TARGETPLATFORM

LABEL maintainer="Dmitry Ponomarev <demdxx@gmail.com>"
LABEL service.name=apfs
LABEL service.weight=1
LABEL service.port=8080
LABEL service.check.http=http://{{address}}/health
LABEL service.check.interval=5s
LABEL service.check.timeout=2s
LABEL service.public=false

RUN apt-get update \
 && apt-get install -y imagemagick curl docker.io \
 && apt-get clean \
 && update-ca-certificates

ENV STORAGE_PROCEDURE_DIR=/procedures
ENV STORAGE_CONVERTERS=image,procedure,shell,exec,docker
ENV WORKER_TAGS=image,gpu,cpu,docker,video,any

RUN mkdir -p /tmp/data/
ADD deploy/procedures /procedures
ADD .build/${TARGETPLATFORM}/apfs /apfs

ENTRYPOINT [ "/apfs" ]
CMD [ "server", "--processing=1" ]
