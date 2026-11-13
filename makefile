.PHONY: install-client wasm build-web run
 
# Install the client dependencies
install-client:
	cd client && bun install
 
# Build crypto.wasm and wasm_exec.js into client/public
wasm:
	bash wasm/run.sh
 
# Build the client and copy it into the server
build-web: install-client wasm
	cd client && bun run build
	rm -rf server/internal/web/dist/*
	cp -R client/dist/. server/internal/web/dist/
	touch server/internal/web/dist/.gitkeep
 
# Build everything and start the server
# Needs JWT_SECRET and AUTH_HASH_KEY to be set (see README)

run: build-web
	cd server && go run ./cmd/server