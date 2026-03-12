# RTSPanda — single-binary build (frontend + backend)
# Run from repo root.

BINARY_NAME := rtspanda
FRONTEND_DIR := frontend
BACKEND_DIR := backend
WEB_EMBED_DIR := $(BACKEND_DIR)/internal/api/web

.PHONY: build build-frontend build-backend clean docker-up docker-down

build: build-frontend build-backend

# Build React app then copy into backend for embedding
build-frontend:
	cd $(FRONTEND_DIR) && npm run build
	@# Replace web with built assets; keep .gitkeep for repo
	rm -rf $(WEB_EMBED_DIR)
	mkdir -p $(WEB_EMBED_DIR)
	cp -r $(FRONTEND_DIR)/dist/. $(WEB_EMBED_DIR)/
	touch $(WEB_EMBED_DIR)/.gitkeep

# Build Go binary (requires build-frontend first, or web/ will be empty)
build-backend:
	cd $(BACKEND_DIR) && go build -o $(BINARY_NAME) ./cmd/rtspanda

clean:
	rm -rf $(FRONTEND_DIR)/dist
	rm -rf $(WEB_EMBED_DIR)
	rm -f $(BACKEND_DIR)/$(BINARY_NAME) $(BACKEND_DIR)/$(BINARY_NAME).exe

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
