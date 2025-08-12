include Makefile.openapi

# Модульный путь
MODULE := $(shell go list -m -f '{{.Path}}')

# Пакеты без cmd/* и internal/gen/*
PKGS := $(shell go list ./... | grep -v '^$(MODULE)/cmd/' | grep -v '^$(MODULE)/internal/gen')

# coverpkg — все выбранные пакеты через запятую
COVERPKG := $(shell echo $(PKGS) | tr ' ' ',')

.PHONY: create
create:
	migrate create -ext sql -dir migrations -seq $(name)

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml ./...

.PHONY: lint-fix
lint-fix:
	golangci-lint run --config .golangci.yml --fix ./...

.PHONY: tests
tests:
	@echo ">> running tests (race, shuffle, coverage) excluding cmd/* and internal/gen/*"
	@mkdir -p coverage
	@rm -f coverage/cover.out
	@bash -lc '\
first=1; \
printf "\n%-45s | %8s | %s\n" "PACKAGE" "COVERAGE" "DURATION"; \
printf "%-45s-+-%8s-+-%s\n" "---------------------------------------------" "--------" "----------------"; \
for pkg in $(PKGS); do \
  profile="coverage/$$(echo $$pkg | tr "/." "__").out"; \
  start=$$(date +%s); \
  go test -race -shuffle=on -coverpkg=$(COVERPKG) -coverprofile=$$profile -count=1 $$pkg >coverage/$$(echo $$pkg | tr "/." "__").log 2>&1; \
  status=$$?; \
  end=$$(date +%s); dur_ms=$$(( (end-start)*1000 )); dur="$${dur_ms}ms"; \
  if [ $$status -ne 0 ]; then \
    printf "%-45s | %8s | %s\n" "$$pkg" "FAILED" "$$dur"; \
    cat coverage/$$(echo $$pkg | tr "/." "__").log; \
    exit $$status; \
  fi; \
  pct=$$(go tool cover -func=$$profile | tail -n 1 | awk "{print \$$3}"); \
  printf "%-45s | %8s | %s\n" "$$pkg" "$$pct" "$$dur"; \
  if [ $$first -eq 1 ]; then \
    cat $$profile > coverage/cover.out; first=0; \
  else \
    tail -n +2 $$profile >> coverage/cover.out; \
  fi; \
done; \
echo; \
echo ">> coverage summary"; \
go tool cover -func=coverage/cover.out | tail -n 1 \
'