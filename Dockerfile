FROM golang:1.24-bookworm

RUN apt-get update \
 && apt-get install -y --no-install-recommends build-essential gcc libc6-dev git make \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src

# Default: keep container alive for interactive `compose exec`
CMD ["sleep", "infinity"]
