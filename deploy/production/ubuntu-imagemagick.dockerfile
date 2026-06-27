FROM ubuntu:plucky

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

RUN apt-get update \
 && apt-get install -y imagemagick curl python3 python3-pip \
 && apt-get clean

COPY deploy/procedures/requirements.txt /tmp/procedures-requirements.txt
# PyTorch CPU wheels: linux/amd64 and linux/arm64 only (no 32-bit ARM).
# On arm/v6|v7 install OpenCV/Pillow for facemeta; image2vec needs torch.
RUN set -eux; \
    case "${TARGETPLATFORM}" in \
      linux/arm/v6|linux/arm/v7) \
        echo "Skipping PyTorch on ${TARGETPLATFORM} (no CPU wheels); image2vec unavailable"; \
        pip3 install --break-system-packages \
          'opencv-python-headless>=4.8,<5' 'Pillow>=10,<12'; \
        ;; \
      *) \
        pip3 install --break-system-packages \
          --index-url https://download.pytorch.org/whl/cpu \
          torch torchvision; \
        pip3 install --break-system-packages -r /tmp/procedures-requirements.txt; \
        ;; \
    esac; \
    rm -f /tmp/procedures-requirements.txt

COPY .build/zoneinfo.zip /usr/local/go/lib/time/
COPY .build/${TARGETPLATFORM}/apfs /
COPY .build/.empty /data
COPY deploy/procedures /procedures

ENTRYPOINT ["/apfs"]

CMD ["server", "--processing=1"]
