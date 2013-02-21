from collections import defaultdict
import csv
from cStringIO import StringIO
import glob
import hashlib
import os
import os.path
import re
import subprocess
import sys

import pybcbgo.flags as flags

__exp_dir = None
__results_dir = None

def set_exp_dir(exp_dir):
    global __exp_dir
    __exp_dir = exp_dir
    
    makedirs(__exp_dir)


def __assert_exp_dir():
    if __exp_dir is None:
        eprintln('Please call "set_exp_dir" before using pybcbgo.')
        exit(1)


def set_results_dir(results_dir):
    global __results_dir
    __results_dir = results_dir
    
    makedirs(__results_dir)


def __assert_results_dir():
    if __results_dir is None:
        eprintln('Please call "set_results_dir" before using pybcbgo.')
        exit(1)


def makedirs(p):
    if not readable(p):
        os.makedirs(p)
    if not readable(p):
        eprintln('Could not create directory "%s".', p)
        exit(1)


def eprint(s):
    print >> sys.stderr, s,


def eprintln(s):
    print >> sys.stderr, s


def veprintln(s):
    if flags.config.verbose:
        eprintln(s)


def exit(n):
    sys.exit(n)


def make():
    cmd('make')
    cmd('make', 'install-exp')


def readable(f):
    return os.access(f, os.R_OK)


def parse_range(s):
    '''
    Takes a range of the form `^(\d+)-(\d+)$` and returns a corresponding
    `xrange` object.

    If no hypen is found in `s`, then `xrange(int(s), int(s)+1)` is returned.

    Any other format is an error.
    '''
    if '-' not in s:
        try:
            n = int(s)
        except ValueError:
            eprintln('Could not parse range "%s".' % s)
            exit(1)
        return xrange(n, n+1)

    mg = re.match('^(\d+)-(\d+)$', s)
    if mg is None:
        eprintln('Could not parse range "%s".' % s)
        exit(1)

    return xrange(int(mg.group(1)), int(mg.group(2)) + 1)


def cached(files, fun):
    '''
    Executes `fun` if and only if its output doesn't already exist.

    The output is defined by a list of files given by `files`. If *any*
    file in `files` does not exist, then `fun` is executed. If `files` is
    empty, then `fun` is executed.

    If `config.ignore_cache` is true, `fun` is executed.

    If `files` contains *any* files with a suffix in `config.no_cache`,
    then `fun` is executed.

    Also, after `fun` is executed, this will check to make sure all of the
    files we think should exist actually do.
    '''
    def assert_fun():
        fun()
        for f in files:
            if not readable(f):
                eprintln('We expected that "%s" exists, but it does not.' % f)
                exit(1)

    if not flags.used('ignore-cache') or not flags.used('no-cache'):
        eprintln('In order to use caching capabilities, the ignore-cache '+
                 'and no-cache flags need to be used in experiment setup.')
        exit(1)

    if flags.config.ignore_cache:
        assert_fun()
        return

    nocache = False
    for f in files:
        if any(map(f.endswith, flags.config.no_cache)):
            nocache = True
            break
    if nocache:
        assert_fun()
        return

    # If any of the files are not readable, execute fun.
    if len(files) == 0 or any(map(lambda f: not readable(f), files)):
        assert_fun()
        return

    veprintln('Using cached files: %s' % ' '.join(files))


def cached_cmd(files, *args, **kwargs):
    def fun():
        cmd(*args, **kwargs)
    cached(files, fun)

def cmd(*args, **kwargs):
    kwargs['stderr'] = subprocess.STDOUT 
    try:
        veprintln(' '.join(args))
        out = subprocess.check_output(args, **kwargs)
        return out
    except Exception, e:
        eprintln('Could not execute command "%s" (exit status: %d)\n\n%s'
                 % (' '.join(args), e.returncode, e.output))
        exit(1)

def pdb_path(pdbid):
    if len(pdbid) not in (4, 5):
        eprintln('Unrecognized PDB identifier: %s' % pdbid)
        exit(1)

    group = pdbid[1:3]
    fname = 'pdb%s.ent.gz' % pdbid[0:4] # chops off chain ID if it exists
    return os.path.join(flags.config.pdb_dir, group, fname)

def pdb_chain(pdbid):
    if len(pdbid) == 5:
        return pdbid[4]
    return None


def pdb_case(pdbid):
    if len(pdbid) not in (4, 5):
        eprintln('Unrecognized PDB identifier: %s' % pdbid)
        exit(1)
    if len(pdbid) == 4:
        return pdbid.lower()
    else:
        return pdbid[0:4].lower() + pdbid[4].upper()


def ejoin(*args):
    __assert_exp_dir()
    return os.path.join(__exp_dir, *args)


def rjoin(*args):
    __assert_results_dir()
    return os.path.join(__results_dir, *args)


def eglob(pat):
    return glob.glob(ejoin(pat))


def rglob(pat):
    return glob.glob(rjoin(pat))


def base_ext(file_path, new_ext):
    return re.sub('\.[^.]+$', '.%s' % new_ext, os.path.basename(file_path))


def pdbids_to_fasta(pdbids):
    for pdbid in pdbids:
        pdb_file, chain = pdb_path(pdbid), pdb_chain(pdbid)
        if not readable(pdb_file):
            eprintln('Cannot read file "%s".' % pdb_file)
            exit(1)

        if chain is not None:
            fasta_file = ejoin('%s.fasta' % pdbid)
            cached_cmd([fasta_file],
                       'pdb2fasta',
                       '--chain', chain, '--separate-chains',
                       '--split', __exp_dir, pdb_file,)
        else:
            cached_cmd(tglob('%s*.fasta' % pdbid),
                       'pdb2fasta', '--separate-chains', '--split',
                       __exp_dir, pdb_file)

def fastas_to_fmap(fastas):
    for fasta in fastas:
        fmap_file = ejoin(base_ext(fasta, 'fmap'))

        args = [
           'hhfrag-map',
           '--cpu', str(flags.config.cpu),
           '--seq-db', flags.config.seq_hhm_db,
           '--pdb-hhm-db', flags.config.pdb_hhm_db,
           '--hhfrag-inc', str(flags.config.hhfrag_inc),
           '--hhfrag-min', str(flags.config.hhfrag_min),
           '--hhfrag-max', str(flags.config.hhfrag_max),
        ]
        if not flags.config.blits:
            args.append('--blits=false')
        args += [fasta, fmap_file]

        cached_cmd([fmap_file], *args)

def fastas_to_fmap_parallel(fastas):
    args = [
       'hhfrag-map-many',
       '--cpu', str(min(6, flags.config.cpu)),
       '--seq-db', flags.config.seq_hhm_db,
       '--pdb-hhm-db', flags.config.pdb_hhm_db,
       '--hhfrag-inc', str(flags.config.hhfrag_inc),
       '--hhfrag-min', str(flags.config.hhfrag_min),
       '--hhfrag-max', str(flags.config.hhfrag_max),
    ]
    if not flags.config.blits:
        args.append('--blits=false')
    args += [__exp_dir] + fastas

    cached_cmd(map(lambda f: ejoin(base_ext(f, 'fmap')), fastas), *args)

def search_bowdb_pdb(prot_files, bow_db=None, chain='',
                     limit=100, min=0.0, max=1.0):
    flags.assert_flag('bow-db')

    if bow_db is None:
        bow_db = flags.config.bow_db
    else:
        bow_db = ejoin(bow_db)
    if isinstance(prot_files, basestring):
        prot_files = [prot_files]

    file_hash = hashlib.md5(''.join(prot_files)).hexdigest()
    outname = 'bowsearch_limit-%d_min-%f_max-%f_chain-%s_%s' \
        % (limit, min, max, chain, file_hash)
    outname = ejoin(outname)

    def runsearch():
        print >> open(outname, 'w+'), \
            cmd('bowsearch', '-output', 'csv',
                '-limit', '%d' % limit,
                '-min', '%f' % min,
                '-max', '%f' % max,
                '--chain', chain,
                bow_db, *prot_files)
    cached([outname], runsearch)

    print outname

    rows = []
    for row in csv.DictReader(open(outname), delimiter='\t'):
        row['Cosine'] = float(row['Cosine'])
        row['Euclid'] = float(row['Euclid'])
        rows.append(row)
    return rows

def search_bowdb_pdb_many(*args, **kwargs):
    '''
    Takes the same arguments as `search_bowdb_pdb`, but returns the rows
    organized into a dict, with the query PDB ids as keys.
    '''
    results = search_bowdb_pdb(*args, **kwargs)
    d = defaultdict(list)
    for row in results:
        d[row['QueryID']].append(row)
    return d

def mk_bowdb_pdbs(name, pdb_files):
    flags.assert_flags('frag-lib')
    bowdb = ejoin(name)

    cached_cmd([bowdb],
        'bowmk', '--overwrite', '--cpu', str(flags.config.cpu),
        bowdb, flags.config.frag_lib, *pdb_files)


def mk_bowdb(name, protein_files):
    flags.assert_flags('frag-lib', 'seq-hhm-db', 'pdb-hhm-db',
                       'hhfrag-inc', 'hhfrag-min', 'hhfrag-max', 'blits')

    bowdb = ejoin(name)

    args = [
       'bowmk',
       '--overwrite', # let the `cached_cmd` work out the deets
       '--cpu', str(flags.config.cpu),
       '--seq-db', flags.config.seq_hhm_db,
       '--pdb-hhm-db', flags.config.pdb_hhm_db,
       '--hhfrag-inc', str(flags.config.hhfrag_inc),
       '--hhfrag-min', str(flags.config.hhfrag_min),
       '--hhfrag-max', str(flags.config.hhfrag_max),
    ]
    if not flags.config.blits:
        args.append('--blits=false')
    args += [bowdb, flags.config.frag_lib] + protein_files

    cached_cmd([bowdb], *args)

