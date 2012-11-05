all: install gofmt data/fraglibs/centers400_11

install:
	go install -p 6 ./fragbag ./matt ./pdb ./rmsd
	go install -p 6 ./cmd/*

gofmt:
	gofmt -w */*.go cmd/*/*.go */example/*/*.go experiments/cmd/*/*.go
	colcheck */*.go cmd/*/*.go */example/*/*.go experiments/cmd/*/*.go

data/fraglibs/%: data/fraglibs/%.brk
	scripts/translate-fraglib "data/fraglibs/$*.brk" "data/fraglibs/$*"

# Utilities
push:
	git push origin master
	git push tufts master
	git push github master

tags:
	find ./ \( \
			-name '*.go' \
			-and -not -wholename './examples/*' \
		\) -print0 \
		| xargs -0 gotags > TAGS

loc:
	find ./ -name '*.go' -print | sort | xargs wc -l

# Experiments with default parameters
exp-fragbag-pride: data/fraglibs/centers400_11
	sh experiments/fragbag-pride/run.sh \
		/media/Nightjar/pdb \
		data/fraglibs/centers400_11 \
		data/experiments/fragbag-pride

exp-kolodny-vs-gallant: data/fraglibs/centers400_11
	sh experiments/kolodny-vs-gallant/run.sh \
		data/experiments/kolodny-vs-gallant/libbuild \
		data/experiments/kolodny-vs-gallant/pdbs \
		data/fraglibs/centers400_11.brk \
		data/fraglibs/centers400_11

exp-bow-vs-matt: data/fraglibs/centers400_11
	sh experiments/bow-vs-matt/run.sh \
		data/experiments/bow-vs-matt/pdbs \
		data/fraglibs/centers400_11 \
		data/experiments/bow-vs-matt/cath-bowdb

