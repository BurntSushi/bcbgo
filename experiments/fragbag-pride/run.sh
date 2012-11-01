function msg {
  echo $* >&2
}

if [ $# != 3 ]; then
  msg "Usage: `basename $0` pdb-dir fraglib-dir pride-data-dir"
  exit 1
fi

exp_dir="experiments/fragbag-pride"

pdb_dir=$1
frag_dir=$2
data_dir=$3

pride_pdb_dir="$data_dir/pdb"
pride_bow_dir="$data_dir/bowdbs"
pride_pair_dir="$data_dir/pair-dists"

# Start fresh
msg "Clearing current data."
rm -rf $data_dir/*
mkdir -p "$data_dir"
mkdir $pride_pdb_dir
mkdir $pride_bow_dir
mkdir $pride_pair_dir

# Make sure all our binaries are up to date
msg "Installing binaries"
make install

msg "Building test executables"
go build -o experiments/cmd/fragbag-ordering/fragbag-ordering \
  ./experiments/cmd/fragbag-ordering

msg "Gathering the pride data set from the PDB"
$exp_dir/gather-pride-dataset --pdb-dir "$pdb_dir" --output-dir "$pride_pdb_dir"

msg "Creating BOW databases from each PDB entry"
$exp_dir/create-pride-bowdbs "$frag_dir" "$pride_pdb_dir" "$pride_bow_dir"

msg "Computing distances between all pairs" 
$exp_dir/generate-pair-dists "$pride_bow_dir" "$pride_pair_dir"

msg "Computing statistics (saved to $exp_dir/results)"
$exp_dir/compute-stats "$pride_pair_dir"/* > $exp_dir/results

msg "Cleanup"
rm experiments/cmd/fragbag-ordering/fragbag-ordering

