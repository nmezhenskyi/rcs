.PHONY: build dev run test tidy clean cleanproto cleanall

PROTO_IN_DIR := api/protobuf
PROTO_OUT_DIR := internal/genproto
PROTOS := $(wildcard $(PROTO_IN_DIR)/*.proto)

genproto: $(PROTOS)
	mkdir -p $(PROTO_OUT_DIR)
	protoc --proto_path=$(PROTO_IN_DIR) \
		--go_out=$(PROTO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(PROTO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTOS)

build: genproto
	go build -o ./bin/rcs ./cmd

dev:
	go run ./cmd

run: build
	./bin/rcs

test:
	go test ./internal/**

tidy:
	go mod tidy

clean:
	-@rm -r ./bin 2>/dev/null || true

cleanproto:
	-@rm -r ./internal/genproto 2>/dev/null || true

cleanall: clean cleanproto

.DEFAULT_GOAL := run
