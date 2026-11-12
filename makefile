build-web:
	cd client && bun run build
	rm -rf server/internal/web/dist/*
	cp -R client/dist/. server/internal/web/dist/
	touch server/internal/web/dist/.gitkeep

run: build-web
	cd server && go run ./cmd/server