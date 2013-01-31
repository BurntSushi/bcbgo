import glob
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
                       'pdb2fasta', '--chain', chain, pdb_file, fasta_file)
        else:
            cached_cmd(tglob('%s*.fasta' % pdbid),
                       'pdb2fasta', '--separate-chains', '--split',
                       __exp_dir, pdb_file)

def fastas_to_fmap(fastas):
    for fasta in fastas:
        fmap_name = re.sub('fasta$', 'fmap', os.path.basename(fasta))
        fmap_file = ejoin(fmap_name)

        args = [
           'hhfrag-map',
           '--cpu', str(flags.config.cpu),
           '--seqdb', flags.config.seq_hhm_db,
           '--pdbdb', flags.config.pdb_hhm_db,
           '--win-inc', str(flags.config.hhfrag_inc),
           '--win-min', str(flags.config.hhfrag_min),
           '--win-max', str(flags.config.hhfrag_max),
        ]
        if flags.config.blits:
            args.append('--blits')
        args += [fasta, fmap_file]

        cached_cmd([fmap_file], *args)

