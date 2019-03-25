all:
	@mkdir -p bin/
	@echo "==> Installing dependencies"
	@go get -d -v ./...

format:
	@echo "==> Formating project ..."
	go fmt ./...

build:
	@echo "==> Building ..."
	@go build -o bin/nestor .

clean:
	@rm nestor

test:
	@echo "==> Testing nestor ..."
	@go list -f '{{range .TestImports}}{{.}} {{end}}' ./... | xargs -n1 go get -d
	go test ./...

PHONY: all format test
