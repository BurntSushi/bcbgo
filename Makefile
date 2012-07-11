all: gofmt data/fraglibs/centers400_11

install:
	go install -p 6 ./matt ./pdb

gofmt:
	gofmt -w */*.go */example/*/*.go
	colcheck */*.go */example/*/*.go

tags:
	find ./ \( \
			-name '*.go' \
			-and -not -wholename './examples/*' \
		\) -print0 \
		| xargs -0 gotags > TAGS

loc:
	find ./ -name '*.go' -print | sort | xargs wc -l

data/fraglibs/%: data/fraglibs/%.brk
	scripts/translate-fraglib "data/fraglibs/$*.brk" "data/fraglibs/$*"

