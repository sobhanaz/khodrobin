.PHONY: help up down logs test lint build fmt eval index

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

up: ## Start the stack
	docker compose up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail all logs
	docker compose logs -f --tail=100

test: ## Run every test suite
	cd services/api && go test -race ./...

lint: ## Vet and format-check
	cd services/api && go vet ./... && test -z "$$(gofmt -l .)"

fmt: ## Format
	cd services/api && gofmt -w .

build: ## Build the API binary
	cd services/api && go build -o bin/api ./cmd/api

index: ## Rebuild the search index from raw data
	cd services/crawler && python build_index.py \
	  --raw ../../data/raw/listings.jsonl --out ../api/data/index.json

# Grades the deployed system, not a library: a parser unit test can pass while
# the endpoint users actually reach is broken. API=... to point elsewhere.
#
# Thresholds differ per set on purpose. The literal and hard sets must stay
# perfect — they are solved, and any drop is a regression. The messy set sits at
# 83.3% because two cases need Persian number-words («پونصد», «نود و پنج»),
# which regexes should not be asked to do. Its threshold is a floor that must
# not fall, and it rises when the model lands.
API ?= https://khodrobin.noxioai.com
eval: ## Grade Persian query understanding against the golden sets
	@set -e; cd services/ai/evals; \
	  python run_eval.py --api $(API) --golden golden_intents.jsonl \
	    --json report_literal.json --threshold 100; \
	  python run_eval.py --api $(API) --golden golden_intents_hard.jsonl \
	    --json report_hard.json --threshold 100; \
	  python run_eval.py --api $(API) --golden golden_intents_messy.jsonl \
	    --json report_messy.json --threshold 83
