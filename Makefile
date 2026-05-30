GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
export PATH := $(GOPATH)/bin:$(PATH)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: api
# generate api proto
api:
	protoc --proto_path=. \
 	       --proto_path=./api \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: proto
# alias for api proto generation
proto: api

.PHONY: test
# run backend tests
test:
	go test ./...

.PHONY: docker-up
# start backend services with Docker Compose
docker-up:
	docker compose up -d --build

.PHONY: docker-down
# stop backend services
docker-down:
	docker compose down

.PHONY: logs-request
# search all lehu Docker logs by request_id
logs-request:
	@test -n "$(RID)" || (echo "Usage: make logs-request RID=<request_id> [SINCE=24h] [TAIL=50000] [CONTEXT=2]" && exit 1)
	bash scripts/find-request-logs.sh "$(RID)" --since "$(or $(SINCE),24h)" --tail "$(or $(TAIL),50000)" --context "$(or $(CONTEXT),2)"

.PHONY: logs-trace
# search all lehu Docker logs by trace_id
logs-trace:
	@test -n "$(TID)" || (echo "Usage: make logs-trace TID=<trace_id> [SINCE=24h] [TAIL=50000] [CONTEXT=2]" && exit 1)
	bash scripts/find-request-logs.sh "$(TID)" --since "$(or $(SINCE),24h)" --tail "$(or $(TAIL),50000)" --context "$(or $(CONTEXT),2)"

.PHONY: logs-search
# search all lehu Docker logs by keyword
logs-search:
	@test -n "$(Q)" || (echo "Usage: make logs-search Q=<keyword> [SINCE=24h] [TAIL=50000] [CONTEXT=2]" && exit 1)
	bash scripts/find-request-logs.sh "$(Q)" --since "$(or $(SINCE),24h)" --tail "$(or $(TAIL),50000)" --context "$(or $(CONTEXT),2)"

.PHONY: smoke
# run local backend smoke checks
smoke:
	bash scripts/smoke.sh

.PHONY: release-check
# run release preflight checks
release-check:
	bash scripts/release-check.sh

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
generate:
	go generate ./...
	go mod tidy

.PHONY: all
# generate all
all:
	make api
	make config
	make generate

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
