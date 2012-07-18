all: install gofmt data/fraglibs/centers400_11

install:
	go install -p 6 ./fragbag ./matt ./pdb ./rmsd
	go install -p 6 ./cmd/*

gofmt:
	gofmt -w */*.go cmd/*/*.go */example/*/*.go
	colcheck */*.go cmd/*/*.go */example/*/*.go

data/fraglibs/%: data/fraglibs/%.brk
	scripts/translate-fraglib "data/fraglibs/$*.brk" "data/fraglibs/$*"

# 'oldstyle' uses fragbag.(*Library).NewBowPDBOldStyle to compute a BOW vector
# for each PDB file when using the fragbag package. This approach computes
# fragments for a flattened list of all ATOM records in each PDB file. This
# results in RMSD calculations that can span over multiple chains.
diff-fragbag-oldstyle:
	GOMAXPROCS=6 diff-kolodny-fragbag \
			--fragbag ~/tmp/collab/libbuild \
			--oldstyle \
			data/fraglibs/centers400_11.brk data/fraglibs/centers400_11 \
			data/kolodny-fragbag-testset/*.pdb

# 'newstyle' uses fragbag.(*Library).NewBowPDB to compute a BOW vector for each 
# PDB file when using the fragbag package. This approach computes fragments for 
# each chain individually, and never computes the RMSD for a set of ATOM 
# records that overlap multiple chains.
diff-fragbag-newstyle:
	GOMAXPROCS=6 diff-kolodny-fragbag \
			--fragbag ~/tmp/collab/libbuild \
			data/fraglibs/centers400_11.brk data/fraglibs/centers400_11 \
			data/kolodny-fragbag-testset/*.pdb

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

