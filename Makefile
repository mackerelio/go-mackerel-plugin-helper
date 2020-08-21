.PHONY: deps
deps:
	go install golang.org/x/lint/golint

.PHONY: lint
lint: deps
	golint -set_exit_status .

.PHONY: test
test:
	go test -v
