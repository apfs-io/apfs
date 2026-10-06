FROM ghcr.io/apfs-io/apfs:ubuntu-media-deps-latest

ARG TARGETPLATFORM
ARG BUILDPLATFORM

LABEL maintainer="GeniusRabbit (Dmitry Ponomarev github.com/demdxx)"
LABEL service.name=apfs
LABEL service.weight=1

ENV LOG_LEVEL=info
ENV STORAGE_METADB_CONNECT=badger:///data/apfs.bdb
ENV STORAGE_STATE_CONNECT=memory
ENV STORAGE_PROCEDURE_DIR=/procedures
ENV STORAGE_CONVERTERS=image,procedure,shell,exec
ENV WORKER_TAGS=video,image,ffmpeg,imagemagic
ENV WORKFLOWS_DIR=/workflows

COPY .build/zoneinfo.zip /usr/local/go/lib/time/
COPY .build/${TARGETPLATFORM}/apfs /
COPY .build/.empty /data
COPY deploy/procedures /procedures

ENTRYPOINT ["/apfs"]

CMD ["server", "--processing=1"]
