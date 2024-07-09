# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
BINARY_NAME=prover_monitor

# Arguments for the program
PUSHGATEWAY_URL=http://your_pushgateway_address:9091
API_BASE_URL=http://your_api_address:8088
ADDRESSES=aleo1ul89ek6egwjtljy6yhmyteyu9y077ruahwggzfh6sgjqp890y5xs6mz9pe,aleo1zp00ltnw23uvdq4spxax3zp84mt7pkvgyerlukxk5t443k6f5v9s9wem4l
INTERVAL=10m

all: test build

build:
	$(GOBUILD) -o $(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run: build
	./$(BINARY_NAME) --pushgateway=$(PUSHGATEWAY_URL) --api=$(API_BASE_URL) --addresses=$(ADDRESSES) --interval=$(INTERVAL)

fmt:
	$(GOFMT) ./...

vet:
	$(GOVET) ./...

test:
	$(GOTEST) -v ./...

.PHONY: all build clean run fmt vet test
