.PHONY: build
build:
	go build -o ./bin/rcs ./cmd

.PHONY: genproto
genproto:
	mkdir -p internal/genproto
	protoc --proto_path=api/protobuf \
		--go_out=internal/genproto --go_opt=paths=source_relative \
		--go-grpc_out=internal/genproto --go-grpc_opt=paths=source_relative \
		rcs.proto

.PHONY: dev
dev:
	go run ./cmd

.PHONY: run
run:
	./bin/rcs

.PHONY: test
test:
	go test ./internal/**

.PHONY: clean
clean:
	-@rm -r ./bin 2>/dev/null || true

.PHONY: cleanproto
cleanproto:
	-@rm -r ./internal/genproto 2>/dev/null || true

.PHONY: compile
compile: build run

.DEFAULT_GOAL := compile
