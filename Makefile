# swallow-ap-hk07-firmware-tools — build & test entry points.
# One repo, three languages: Rust (quarry), Go (swallow), Zig (lure).
.DEFAULT_GOAL := help
.PHONY: help ci hooks test test-race lint fmt fmt-check cover \
        test-rust test-go test-zig test-firmware lint-rust lint-go lint-zig \
        dist tui screenshots clean

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'

## ci: everything CI runs — format check, lint, tests (race), all three langs
## This is the project's ONLY CI: there is no hosted CI (no GitHub Actions).
ci: fmt-check lint test-race
	@echo "== CI OK =="

## hooks: enable the local pre-push CI gate (runs `make ci` before every push)
hooks:
	git config core.hooksPath githooks
	@echo "local CI hook enabled: git will run `make ci` before each push"

## test: run every test suite (Rust + Go + Zig unit + lure integration)
test: test-rust test-go test-zig

## test-race: like `test`, with the Go race detector
test-race: test-rust test-zig
	cd go && go test -race ./...

test-rust:
	cargo test --workspace

## test-firmware: validate the header parser against real images (set FW=<dir>)
test-firmware:
	QUARRY_FIRMWARE_DIR=$(or $(FW),$$HOME/Downloads) \
	  cargo test -p quarry --test real_images -- --nocapture
test-go:
	cd go && go test ./...
test-zig:
	cd zig && zig build test
	cd zig && ./tftp_test.sh

## lint: static analysis across all three languages (warnings are errors)
lint: lint-rust lint-go lint-zig
lint-rust:
	cargo clippy --workspace --all-targets -- -D warnings
lint-go:
	cd go && go vet ./...
lint-zig:
	cd zig && zig build   # zig's compiler is the linter

## fmt: auto-format all sources
fmt:
	cargo fmt
	cd go && gofmt -w .
	cd zig && zig fmt lure.zig build.zig

## fmt-check: fail if anything is unformatted (CI gate)
fmt-check:
	cargo fmt --check
	@out="$$(cd go && gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi
	cd zig && zig fmt --check lure.zig build.zig

## cover: Go coverage summary (per-package + total)
cover:
	cd go && go test ./... -coverprofile=/tmp/swallow.cover >/dev/null && \
	  go tool cover -func=/tmp/swallow.cover | tail -1

## dist: cross-compiled release binaries + SHA256SUMS into dist/
dist:
	./scripts/dist.sh

## tui: run the dashboard
tui:
	cd go && go run ./cmd/swallow

## screenshots: regenerate README screenshots via the termwright harness
screenshots:
	cd go && go build -o /tmp/swallow ./cmd/swallow
	cd tools/tui-harness && SWALLOW_BIN=/tmp/swallow SHOT_DIR=$(CURDIR)/docs/screenshots cargo run

## clean: remove build artifacts
clean:
	cargo clean
	rm -rf dist zig/zig-out zig/.zig-cache tools/tui-harness/target
