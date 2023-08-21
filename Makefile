ifdef ComSpec
	EXE := .exe
endif

build:
	go build -o bin/vectopng$(EXE) perron2.ch/vectopng

format:
	go fmt ./...

cost:
	scc
