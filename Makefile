deps:
	go get -d -v -t .
	go get github.com/golang/lint/golint

lint: deps
	go tool vet -all .
	golint -set_exit_status .

test: lint
	go test -v
