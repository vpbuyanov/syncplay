include Makefile.openapi

.PHONY: create
create:
	migrate create -ext sql -dir migrations -seq $(name)

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --config .golangci.yml --fix ./...
