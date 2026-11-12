# 1. wasm
FROM --platform=$BUILDPLATFORM golang:1.27 AS wasm
WORKDIR /src
COPY wasm/ ./wasm/
WORKDIR /src/wasm
RUN mkdir -p /out && \
    GOOS=js GOARCH=wasm go build -o /out/crypto.wasm . && \
    if [ -f "$(go env GOROOT)/lib/wasm/wasm_exec.js" ]; then \
    cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/; \
    else \
    cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" /out/; \
    fi

# 2. client
FROM --platform=$BUILDPLATFORM oven/bun:1 AS client
WORKDIR /app
COPY client/package.json client/bun.lock ./
RUN bun install --frozen-lockfile
COPY client/ ./
COPY --from=wasm /out/ ./public/
RUN bun run build

# 3. server
FROM --platform=$BUILDPLATFORM golang:1.27 AS server
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY server/ ./server/
COPY --from=client /app/dist ./server/internal/web/dist

WORKDIR /src/server

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o /out/server ./cmd/server

# 4. runtime
FROM gcr.io/distroless/static:nonroot
COPY --from=server /out/server /server
ENV HOST=0.0.0.0 PORT=8090
EXPOSE 8090
ENTRYPOINT ["/server"]