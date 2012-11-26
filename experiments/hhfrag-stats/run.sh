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
  msg "Usage: `basename $0` [--blits | --cpu n] pdb-dir pdb-hhm-db seq-hhm-db targets"
  exit 1
}

blits=""
num_cpus=`num_cpus`
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

if [ $# != 4 ]; then
  usage
fi

exp_dir="experiments/hhfrag-stats"
data_dir="data/experiments/hhfrag-stats"
tmp_dir="data/experiments/hhfrag-stats/tmp"
calc_stats=experiments/cmd/hhfrag-stats/hhfrag-stats

pdb_dir="$1"
pdb_hhm_db="$2"
seq_hhm_db="$3"
targets="$4"
log_path=$exp_dir/"$(basename "$4")"
map_dir=$tmp_dir/map

if [ ! -f "$targets" ]; then
  msg "Could not read $targets"
  exit 1
fi

# Make sure all our binaries are up to date
msg "Installing binaries"
make install
make install-tools

msg "Building test executables"
go build -o $calc_stats ./experiments/cmd/hhfrag-stats

if [ -z "$blits" ]; then
  prefix="$seq_hhm_db-$pdb_hhm_db-hhsearch"
else
  prefix="$seq_hhm_db-$pdb_hhm_db-hhblits"
fi

mkdir -p "$log_path"
mkdir -p "$tmp_dir"
mkdir -p "$map_dir"

if [ -f "$log_path/$prefix" ]; then
  echo "$log_path already exists; skipping experiment"
else
  mkdir -p "$log_path/$prefix"
  mkdir -p "$map_dir/$prefix"

  rm "$tmp_dir"/*.fasta
  for target in $(cat "$targets"); do
    case ${#target} in
      4)
        pdb_file="$pdb_dir"/${target:1:2}/pdb$target.ent.gz
        pdb2fasta --seqres --separate-chains --split "$tmp_dir" "$pdb_file"
        ;;
      5)
        pdbid=${target:0:4}
        chain=${target:4}
        pdb_file="$pdb_dir"/${pdbid:1:2}/pdb$pdbid.ent.gz
        pdb2fasta --chain $chain --seqres "$pdb_file" "$tmp_dir"/$target.fasta
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
        $blits \
        "$target" > "$map_dir/$prefix/$name.fmap"
    fi
  done

  for fmap in "$map_dir/$prefix"/*.fmap; do
    name=$(basename "${fmap%*.fmap}")
    $calc_stats "$fmap" > "$log_path/$prefix/$name.log"
  done
fi

msg "Cleanup"
rm $calc_stats

