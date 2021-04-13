default: test

test:
	go test ./...

bench:
	go test ./... -run=NONE -bench=. -benchmem

staticcheck:
	staticcheck ./...

doc: README.md omhttp/README.md

README.md: README.md.tpl $(wildcard *.go)
	becca -package github.com/bsm/openmetrics

omhttp/README.md: omhttp/README.md.tpl $(wildcard *.go)
	cd omhttp; becca -package github.com/bsm/openmetrics/omhttp
