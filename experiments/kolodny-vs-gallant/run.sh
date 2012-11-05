function msg {
  echo $* >&2
}

if [ $# != 4 ]; then
  msg "Usage: `basename $0` old-fragbag some-pdb-dir fraglib-file fraglib-dir"
  exit 1
fi

exp_dir="experiments/kolodny-vs-gallant"
differ="experiments/cmd/diff-kolodny-fragbag/diff-kolodny-fragbag"

old_fragbag=$1
some_pdb_dir=$2
frag_file=$3
frag_dir=$4

# Make sure all our binaries are up to date
msg "Installing binaries"
make install

msg "Building test executables"
go build -o $differ ./experiments/cmd/diff-kolodny-fragbag

if [ ! -f $exp_dir/concat-chains.log ]; then
  $differ \
    --oldstyle \
    --fragbag "$old_fragbag" \
    "$frag_file" \
    "$frag_dir" \
    "$some_pdb_dir"/* > $exp_dir/concat-chains.log
fi

if [ ! -f $exp_dir/separate-chains.log ]; then
  $differ \
    --fragbag "$old_fragbag" \
    "$frag_file" \
    "$frag_dir" \
    "$some_pdb_dir"/* > $exp_dir/separate-chains.log
fi

msg "Cleanup"
rm $differ

