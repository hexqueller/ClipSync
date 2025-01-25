.SILENT:

all:
	go mod download
	CGO_ENABLED=0 GOOS=linux go build -o clipsync ./main.go

clean:
	rm -f clipsync

.PHONY: all clean