FROM scratch

ARG TARGETPLATFORM
ARG BUILDPLATFORM

LABEL maintainer="GeniusRabbit (Dmitry Ponomarev github.com/demdxx)"
LABEL service.name=apfs
LABEL service.weight=1

ENV LOG_LEVEL=info
ENV STORAGE_METADB_CONNECT=sqlite3:///data/apfs.db?cache=shared
ENV STORAGE_STATE_CONNECT=memory
ENV STORAGE_PROCEDURE_DIR=/procedures
ENV STORAGE_CONVERTERS=image,procedure,shell,exec,docker
ENV WORKER_TAGS=image,gpu,cpu,docker,video,any
ENV WORKFLOWS_DIR=/workflows

COPY .build/zoneinfo.zip /usr/local/go/lib/time/
COPY .build/${TARGETPLATFORM}/apfs /
COPY .build/.empty /data
COPY deploy/procedures /procedures

ENTRYPOINT ["/apfs", "server", "--processing=1"]
