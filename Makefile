.PHONY: test tui screenshots
test:
	cargo test --workspace
	cd go && go test ./...
	cd zig && zig build
tui:
	cd go && go run ./cmd/swallow
# regenerate README screenshots (builds swallow, drives the TUI with termwright)
screenshots:
	cd go && go build -o /tmp/swallow ./cmd/swallow
	cd tools/tui-harness && SWALLOW_BIN=/tmp/swallow SHOT_DIR=$(CURDIR)/docs/screenshots cargo run
