.PHONY: all clean
# 被编译的文件
BUILDFILE=main.go
# 编译后的静态链接库文件名称
TARGETNAME=ats_check
# GOOS为目标主机系统 
# mac os : "darwin"
# linux  : "linux"
# windows: "windows"
GOOS=linux
# GOARCH为目标主机CPU架构, 默认为amd64 
GOARCH=amd64

all: format test build clean

test:
	go test -v . 

format:
	gofmt -w .

build:
	rm -rf releases/$(TARGETNAME).tar.gz
	mkdir -p releases
	cp atscheck.sh releases/atscheck.sh
	cp -r config releases/
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -v -o releases/$(TARGETNAME) $(BUILDFILE)
	cp releases/$(TARGETNAME) .
	tar -cf releases/$(TARGETNAME).tar.gz releases/atscheck.sh releases/config releases/$(TARGETNAME)
	rm -rf releases/atscheck.sh releases/config releases/$(TARGETNAME)
	cp releases/$(TARGETNAME).tar.gz ~/Downloads/
clean:
	go clean -i