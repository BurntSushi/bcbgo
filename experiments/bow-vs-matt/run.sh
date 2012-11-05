function msg {
  echo $* >&2
}

if [ $# != 3 ]; then
  msg "Usage: `basename $0` some-pdb-dir fraglib-dir bow-dir"
  exit 1
fi

exp_dir="experiments/bow-vs-matt"
differ="experiments/cmd/bow-vs-matt/bow-vs-matt"

some_pdb_dir=$1
frag_dir=$2
bow_dir=$3
log_path=$exp_dir/"$(basename "$bow_dir").log"

# Make sure all our binaries are up to date
msg "Installing binaries"
make install

msg "Building test executables"
go build -o $differ ./experiments/cmd/bow-vs-matt

msg "Generating orderings with fragbag and matt"
msg "This is an all-against-all comparison using each method"
msg "It may therefore take a while"
msg "(Matt may not be able to align all pairs; this is OK)"
if [ ! -f "$log_path" ]; then
  $differ \
    "$bow_dir" \
    "$frag_dir" \
    "$some_pdb_dir"/* > "$log_path"
else
  echo "$log_path already exists; skipping experiment"
fi

msg "Cleanup"
rm $differ

