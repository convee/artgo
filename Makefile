GO ?= go
HOST ?= 127.0.0.1
EXAMPLE_PORT ?= 8080
PORT ?=
EXAMPLE_BIN ?= /tmp/artgo-basic-example

.PHONY: test race vet check bench bench-smoke example smoke

test:
	$(GO) test ./...

race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

check: test race vet

bench:
	$(GO) test -run '^$$' -bench . -benchmem ./examples/performance

bench-smoke:
	$(GO) test -run '^$$' -bench . -benchtime=1x -benchmem ./examples/performance

example:
	cd examples/basic && $(GO) run . $(HOST):$(EXAMPLE_PORT)

smoke:
	@set -eu; \
	port="$(PORT)"; \
	if [ -z "$$port" ]; then port=$$($(GO) run ./examples/port); fi; \
	$(GO) build -o $(EXAMPLE_BIN) ./examples/basic; \
	cd examples/basic; \
	$(EXAMPLE_BIN) "$(HOST):$$port" & \
	pid=$$!; \
	trap 'kill $$pid 2>/dev/null || true; wait $$pid 2>/dev/null || true' EXIT; \
	ready=0; \
	for i in 1 2 3 4 5 6 7 8 9 10; do \
		if curl -fsS "http://$(HOST):$$port/api/health" >/dev/null; then ready=1; break; fi; \
		sleep 0.2; \
	done; \
	test "$$ready" -eq 1; \
	curl -fsS "http://$(HOST):$$port/" | grep -q 'artgo basic example'; \
	curl -fsS "http://$(HOST):$$port/users/alice" | grep -q 'alice'; \
	curl -fsS "http://$(HOST):$$port/template/alice" | grep -q 'ALICE'; \
	curl -fsS "http://$(HOST):$$port/static/app.js" | grep -q 'artgo basic example'; \
	curl -fsS -X POST -H 'Content-Type: application/json' \
		-d '{"name":"Alice","age":30}' "http://$(HOST):$$port/api/users" | grep -q '"name":"Alice"'; \
	echo 'smoke: ok'
