ORGANISATION = ldejager
VERSION ?= latest
COMPONENT = assume-role

.PHONY: build

ifneq ($(shell uname), Darwin)
	EXTLDFLAGS = -extldflags "-static" $(null)
else
	EXTLDFLAGS =
endif

all: deps build

deps:
	go get -u github.com/ldejager/assume-role

build: build_static build_cross build_tar build_sha

build_static:
	go install github.com/ldejager/assume-role
	mkdir -p release
	cp $(GOPATH)/bin/assume-role release/

build_cross:
	GOOS=linux   GOARCH=amd64 CGO_ENABLED=0 go build -o release/linux/amd64/assume-role   github.com/ldejager/assume-role
	GOOS=darwin  GOARCH=amd64 CGO_ENABLED=0 go build -o release/darwin/amd64/assume-role  github.com/ldejager/assume-role

build_tar:
	tar -cvzf release/linux/amd64/assume-role.tar.gz -C release/linux/amd64 assume-role
	tar -cvzf release/darwin/amd64/assume-role.tar.gz -C release/darwin/amd64 assume-role

build_sha:
	sha256sum release/linux/amd64/assume-role.tar.gz > release/linux/amd64/assume-role.sha256
	sha256sum release/darwin/amd64/assume-role.tar.gz > release/darwin/amd64/assume-role.sha256

default: all
