cdir = $(notdir $(CURDIR))
TARGET_LOCAL = ${cdir}.exe
TARGET_LINUX = ${cdir}_linux_amd64
default:
	go build -o ${TARGET_LOCAL}
linux:
	GOOS=linux GOARCH=amd64 go build -o ${TARGET_LINUX}
clean:
	go clean -x -i
	@rm -f ${TARGET_LINUX}