FROM golang:1.23

EXPOSE 8080 8081 8082

ARG TARGETPLATFORM

LABEL maintainer="Dmitry Ponomarev <demdxx@gmail.com>"
LABEL service.name=apfs
LABEL service.weight=1
LABEL service.port=8080
LABEL service.check.http=http://{{address}}/health-check
LABEL service.check.interval=5s
LABEL service.check.timeout=2s
LABEL service.public=false

RUN apt-get update
RUN apt-get install -y imagemagick
RUN apt-get clean
RUN update-ca-certificates

RUN mkdir -p /tmp/data/
ADD deploy/procedures /procedures
ADD .build/${TARGETPLATFORM}/apfs /apfs

ENTRYPOINT [ "/apfs" ]

CMD [ "server", "--processing=1" ]