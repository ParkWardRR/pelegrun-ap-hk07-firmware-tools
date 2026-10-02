# pelegrun-ap-hk07-firmware-tools — build & test entry points.
# Two binaries: quarry (Rust, the core) and pelegrun (Go, TUI + CLI).
.DEFAULT_GOAL := help
.PHONY: help ci hooks test test-race lint fmt fmt-check cover \
        test-rust test-go test-firmware lint-rust lint-go \
        dist tui tour tour-offline clean

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //'

## ci: everything CI runs — format check, lint, tests (race)
## This is the project's ONLY CI: there is no hosted CI (no GitHub Actions).
ci: fmt-check lint test-race
	@echo "== CI OK =="

## hooks: enable the local pre-push CI gate (runs `make ci` before every push)
hooks:
	git config core.hooksPath githooks
	@echo "local CI hook enabled: git will run `make ci` before each push"

## test: run every test suite (Rust + Go)
test: test-rust test-go

## test-race: like `test`, with the Go race detector
test-race: test-rust
	cd go && go test -race ./...

test-rust:
	cargo test --workspace

## test-firmware: validate the header parser against real images (set FW=<dir>)
test-firmware:
	QUARRY_FIRMWARE_DIR=$(or $(FW),$$HOME/Downloads) \
	  cargo test -p quarry --test real_images -- --nocapture
test-go:
	cd go && go test ./...

## lint: static analysis (warnings are errors)
lint: lint-rust lint-go
lint-rust:
	cargo clippy --workspace --all-targets -- -D warnings
lint-go:
	cd go && go vet ./...

## fmt: auto-format all sources
fmt:
	cargo fmt
	cd go && gofmt -w .

## fmt-check: fail if anything is unformatted (CI gate)
fmt-check:
	cargo fmt --check
	@out="$$(cd go && gofmt -l .)"; if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

## cover: Go coverage summary (per-package + total)
cover:
	cd go && go test ./... -coverprofile=/tmp/pelegrun.cover >/dev/null && \
	  go tool cover -func=/tmp/pelegrun.cover | tail -1

## dist: cross-compiled release binaries + SHA256SUMS into dist/
dist:
	./scripts/dist.sh

## tui: run the dashboard
tui:
	cd go && go run ./cmd/pelegrun

## tour: re-record docs/tour.gif from the live TUI with vhs, then optimize
tour:
	cd go && go build -o /tmp/pelegrun ./cmd/pelegrun
	vhs docs/tour.tape
	@command -v magick >/dev/null && magick docs/tour.gif -layers Optimize docs/tour.gif || true

## tour-offline: regenerate docs/tour.gif + screenshots with the pure-Go renderer (no vhs/ffmpeg)
tour-offline:
	cd go && go run ./cmd/tuigif -out $(CURDIR)/docs/tour.gif
	cd go && go run ./cmd/tuigif -shots $(CURDIR)/docs/screenshots

## clean: remove build artifacts
clean:
	cargo clean
	rm -rf dist
