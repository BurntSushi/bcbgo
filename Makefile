all: gofmt

install:
	go install -p 6 ./matt ./pdb

gofmt:
	gofmt -w */*.go */example/*/*.go
	colcheck */*.go */example/*/*.go

tags:
	find ./ \( -name '*.go' -and -not -wholename './examples/*' \) -print0 | xargs -0 gotags > TAGS

loc:
	find ./ -name '*.go' -print | sort | xargs wc -l

ex-%:
	go run examples/$*/main.go

