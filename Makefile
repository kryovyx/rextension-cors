# Per-module checks. Self-contained on purpose: a module cloned on its own
# carries its own `make check` (D9). There is no CI — these targets are the
# only gate, so they have to be runnable from inside a single module.

.PHONY: check fmt vet test cover clean

check: fmt vet test

## fmt — fail if anything is not gofmt-clean (does not rewrite)
fmt:
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

## fmt-fix — rewrite in place
.PHONY: fmt-fix
fmt-fix:
	gofmt -w .

vet:
	go vet ./...

## test — -race is mandatory: the dix defects passed at 100% statement
## coverage because they are concurrency bugs, not branches (§10.2).
test:
	go test -race ./...

cover:
	go test -coverprofile=cover.out ./... && go tool cover -func=cover.out | tail -1

clean:
	rm -f cover.out cover.html
