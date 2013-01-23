set -e

function msg {
  echo $* >&2
}

function num_cpus {
  if [ -r /proc/cpuinfo ]; then
    NPROC=$(cat /proc/cpuinfo | grep '^processor' | wc -l | tr -d ' ')
  else
    NPROC=1
  fi
  echo $NPROC
}

function usage {
  msg "Usage: `basename $0` [--blits | --cpu n | --win-min n | --win-max n | --win-inc 3] bowdb pdb-dir pdb-hhm-db seq-hhm-db targets"
  exit 1
}

blits=""
num_cpus=`num_cpus`
win_min=6
win_max=22
win_inc=3
while true; do
  case "$1" in
    -blits|--blits)
      blits="--blits"
      shift 1
      ;;
    -cpu|--cpu)
      num_cpus=$2
      shift 2
      ;;
    -win-min|--win-min)
      win_min=$2
      shift 2
      ;;
    -win-max|--win-max)
      win_max=$2
      shift 2
      ;;
    -win-inc|--win-inc)
      win_inc=$2
      shift 2
      ;;
    -h|-help|--help)
      usage
      ;;
    -*|--*)
      msg "Invalid flag $1."
      msg
      usage
      ;;
    *)
      break
      ;;
  esac
done

if [ $# != 5 ]; then
  usage
fi

exp_dir="experiments/hhfrag-bow"
data_dir="data/experiments/hhfrag-bow"
tmp_dir="$data_dir/tmp"

bowdb="$1"
pdb_dir="$2"
pdb_hhm_db="$3"
seq_hhm_db="$4"
targets="$5"
log_path=$exp_dir/"$(basename "$targets")"
map_dir=$tmp_dir/map

if [ ! -r "$targets" ]; then
  msg "Could not read $targets"
  exit 1
fi

# Make sure all our binaries are up to date
msg "Installing binaries"
make install
make install-tools

hhsuite_search="hhblits"
if [ -z "$blits" ]; then
  hhsuite_search="hhsearch"
fi
prefix="$seq_hhm_db-$pdb_hhm_db-$hhsuite_search-$win_min-$win_max-$win_inc"

results_bowdb_pdb="$log_path/$prefix/bowdb-pdb.csv"
results_bowdb_hhfrag="$log_path/$prefix/bowdb-hhfrag.csv"

mkdir -p "$log_path"
mkdir -p "$tmp_dir"
mkdir -p "$map_dir"

mkdir -p "$log_path/$prefix"
mkdir -p "$map_dir/$prefix"

rm -rf "$results_bowdb_pdb" "$results_bowdb_hhfrag"
touch "$results_bowdb_pdb" "$results_bowdb_hhfrag"

rm -f "$tmp_dir"/*.fasta
for target in $(cat "$targets"); do
  case ${#target} in
    4)
      pdb_file="$pdb_dir"/${target:1:2}/pdb$target.ent.gz
      bowpdb --csv --limit 100 --quiet \
        "$bowdb" "$pdb_file" >> "$results_bowdb_pdb"
      pdb2fasta --separate-chains --split "$tmp_dir" "$pdb_file"
      ;;
    5)
      pdbid=${target:0:4}
      chain=${target:4}
      pdb_file="$pdb_dir"/${pdbid:1:2}/pdb$pdbid.ent.gz
      bowpdb --csv --limit 100 --quiet --chain $chain \
        "$bowdb" "$pdb_file" >> "$results_bowdb_pdb"
      pdb2fasta --chain $chain "$pdb_file" "$tmp_dir"/$target.fasta
      ;;
    *)
      msg "Unrecognized PDB identifier: $target"
      exit 1
      ;;
  esac
done
for target in "$tmp_dir"/*.fasta; do
  name=$(basename "${target%*.fasta}")
  fmap_file="$map_dir/$prefix/$name.fmap"
  if [ -f "$fmap_file" ]; then
    msg "Skipping $name map generation since $fmap_file exists."
  else
    msg "Computing map for $name..."
    hhfrag-map \
      --cpu $num_cpus \
      --seqdb "$seq_hhm_db" \
      --pdbdb "$pdb_hhm_db" \
      --win-min "$win_min" --win-max "$win_max" --win-inc "$win_inc" \
      $blits \
      "$target" "$map_dir/$prefix/$name.fmap"
  fi
done

for target in "$tmp_dir"/*.fasta; do
  name=$(basename "${target%*.fasta}")
  fmap_file="$map_dir/$prefix/$name.fmap"
  bowseq --csv --limit 100 --quiet \
    "$bowdb" "$fmap_file" >> "$results_bowdb_hhfrag"
done

"$exp_dir/calc-jaccard" \
  "$results_bowdb_pdb" \
  "$results_bowdb_hhfrag" \
  > "$log_path/$prefix/jaccard.csv"

msg "Done."

