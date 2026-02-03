[parallel]
serve: serve-backend serve-frontend

serve-backend:
    air

serve-frontend:
    npm -C frontend run dev

#

test:
    go test ./...

#

lint: lint-go lint-ts

lint-go:
    golangci-lint run

lint-ts:
    npm -C frontend run lint

#

fmt: fmt-go fmt-ts

fmt-go:
    go fmt ./...

fmt-ts:
    npm -C frontend run format

#

local-db:
    cd local/swopcart-dev && docker compose up

start-local-db:
    cd local/swopcart-dev && docker compose up -d

stop-local-db:
    cd local/swopcart-dev && docker compose down
