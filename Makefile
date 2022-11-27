.PHONY: build genproto dev run test cover tidy clean cleanproto cleanall

PROTO_IN_DIR := api/protobuf
PROTO_OUT_DIR := internal/genproto
PROTOS := $(wildcard $(PROTO_IN_DIR)/*.proto)

$(PROTO_OUT_DIR): $(PROTOS)
	mkdir -p $@
	protoc --proto_path=$(PROTO_IN_DIR) \
		--go_out=$@ --go_opt=paths=source_relative \
		--go-grpc_out=$@ --go-grpc_opt=paths=source_relative \
		$(PROTOS)

build: $(PROTO_OUT_DIR)
	go build -o ./bin/rcs ./cmd

genproto: $(PROTO_OUT_DIR)

dev:
	go run ./cmd

run: build
	./bin/rcs

test:
	go test ./internal/**

cover:
	go test -coverprofile cover.out ./internal/**

tidy:
	go mod tidy

clean:
	-@rm -r ./bin 2>/dev/null || true

cleanproto:
	-@rm -r $(PROTO_OUT_DIR) 2>/dev/null || true

cleanall: clean cleanproto

.DEFAULT_GOAL := run
