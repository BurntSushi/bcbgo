all: install gofmt data/fraglibs/centers400_11

install:
	go install -p 6 ./apps/* ./bowdb ./fragbag ./hhfrag ./io/* ./rmsd
	go install -p 6 ./cmd/*

gofmt:
	gofmt -w */*.go */*/*.go cmd/*/*.go */example/*/*.go experiments/cmd/*/*.go
	colcheck */*.go */*/*.go cmd/*/*.go */example/*/*.go experiments/cmd/*/*.go

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

test:
	go test ./rmsd

build-exp:
	go build ./experiments/cmd/*

bench-rmsd:
	go test ./rmsd -run 'nomatch' -bench '.*'

# Experiments with default parameters
exp-hhfrag-stats:
	sh experiments/hhfrag-stats/run.sh \
		/data/bio/pdb \
		kalev \
		nr20 \
		data/experiments/hhfrag-stats/casp9one

exp-fragbag-pride: data/fraglibs/centers400_11
	sh experiments/fragbag-pride/run.sh \
		/data/bio/pdb \
		data/fraglibs/centers400_11 \
		data/experiments/fragbag-pride

exp-kolodny-vs-gallant: data/fraglibs/centers400_11
	sh experiments/kolodny-vs-gallant/run.sh \
		data/experiments/kolodny-vs-gallant/libbuild \
		data/experiments/kolodny-vs-gallant/pdbs \
		data/fraglibs/centers400_11.brk \
		data/fraglibs/centers400_11

exp-bow-vs-matt-cath: data/fraglibs/centers400_11
	sh experiments/bow-vs-matt/run.sh \
		data/experiments/bow-vs-matt/pdbs \
		data/fraglibs/centers400_11 \
		data/experiments/bow-vs-matt/cath-bowdb

exp-bow-vs-matt-za: data/fraglibs/centers400_11
	sh experiments/bow-vs-matt/run.sh \
		/data/bit/pdb/za \
		data/fraglibs/centers400_11 \
		data/experiments/bow-vs-matt/za-bowdb

