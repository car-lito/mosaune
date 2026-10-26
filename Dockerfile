# syntax=docker/dockerfile:1
# Both compilers run on the builder's platform. Only the final Go executable is
# cross-compiled; the runtime image contains neither Node nor a Go toolchain.
FROM --platform=$BUILDPLATFORM node:24-bookworm-slim@sha256:a5f4366e834695d5f363b920c104953a535d3a9864d1bdf74566a0ebe01daee5 AS web
WORKDIR /source
COPY package.json package-lock.json ./
RUN npm ci
COPY web ./web
COPY theme ./theme
COPY scripts ./scripts
COPY vite.config.ts tsconfig.json ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm@sha256:d9c68c2c51161e12fd77e4c6320687c9cd86e1af1e3ad6e6cd63ff970641453c AS build
WORKDIR /source
COPY go.mod go.sum ./
COPY extension-kit/go.mod extension-kit/go.sum ./extension-kit/
RUN go mod download && cd extension-kit && go mod download
COPY . .
COPY --from=web /source/web/dist ./web/dist
COPY --from=web /usr/local/bin/node /usr/local/bin/node
COPY --from=web /source/node_modules/svelte ./node_modules/svelte
COPY --from=web /source/node_modules/esm-env ./node_modules/esm-env
RUN cd extension-kit && CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags="-s -w" -o providers/tmdb/provider.wasm ./providers/tmdb/source
RUN cd extension-kit && CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags="-s -w" -o modules/audiotags/tags.wasm ./modules/audiotags/source
RUN cd extension-kit && for kind in films series music catalogue-films catalogue-series catalogue-music detail-series; do CGO_ENABLED=0 GOOS=wasip1 GOARCH=wasm go build -trimpath -ldflags="-s -w" -o modules/$kind/recognizer.wasm ./modules/$kind/source || exit 1; done
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/mosaune .
RUN mkdir -p build /empty-data && chmod 700 /empty-data && node --input-type=module -e "import {buildNotices} from './scripts/build-notices.mjs'; buildNotices('/source', 'go', process.env)"

FROM scratch
LABEL org.opencontainers.image.title="Mosaune" \
      org.opencontainers.image.source="https://github.com/car-lito/mosaune" \
      org.opencontainers.image.description="Prototype de serveur multimédia personnel — HTTPS direct"
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=build /out/mosaune /mosaune
COPY --from=build /source/build/THIRD-PARTY-NOTICES.txt /THIRD-PARTY-NOTICES.txt
COPY --from=build --chown=65532:65532 /empty-data /data
USER 65532:65532
ENV MOSAUNE_LISTEN=0.0.0.0 MOSAUNE_DATA_DIR=/data
WORKDIR /data
EXPOSE 47831
STOPSIGNAL SIGTERM
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 CMD ["/mosaune", "-healthcheck"]
ENTRYPOINT ["/mosaune"]
