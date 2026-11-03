.PHONY: run build build-windows debug check

run:
	go run . $(ARGS)

build:
	go tool mygo build -skip-dmg

build-windows:
	go run ./scripts/build-windows.go

debug:
	go build -tags e2e -o build/patch .

check:
	go test ./...
	go vet ./...
	go build -o build/patch .
