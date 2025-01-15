FROM golang:1.23

ARG TARGETPLATFORM

LABEL maintainer="Dmitry Ponomarev <demdxx@gmail.com>"
LABEL service.name=apfs-test-app

ADD ./testdata /testdata
ADD .build/${TARGETPLATFORM}/testapp /testapp

CMD [ "/testapp" ]