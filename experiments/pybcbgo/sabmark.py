import csv
import os
import os.path
import sys

import pybcbgo.flags as flags


__TWILIGHT = 'twi_fp'
__SUPERFAMILY = 'sup_fp'


def join(*args):
    flags.assert_flag('sabmark-dir')
    flags.assert_flag('sabmark-set')
    if flags.config.sabmark_set == 'twilight':
        return os.path.join(flags.config.sabmark_dir, __TWILIGHT, *args)
    else:
        return os.path.join(flags.config.sabmark_dir, __SUPERFAMILY, *args)


def gjoin(group_num, *args):
    return join('group%d' % group_num, *args)

def group_exists(group_num):
    return os.access(join('group%d' % group_num), os.R_OK)


def sabid_pdb_path(group, sabid):
    return gjoin(group, 'pdb', sabid + '.ent')


def group(group_num):
    '''
    `group_num` is the group number in the alignment set of the context.
    (The alignment set is discovered through the `sabmark-set` flag.)

    Returns a tuple of lists, where the first is a list of true positive
    SCOP identifiers and the second is a list of false positive SCOP
    identifiers in the group specified.
    '''
    if not group_exists(group_num):
        print >> sys.stderr, 'SABmark group %d does not exist in %s' \
            % (group_num, flags.config.sabmark_set)
        sys.exit(1)

    tps, fps = [], []
    gsummary = open(join('group%d' % group_num, 'group.summary'))
    for row in csv.DictReader(gsummary, delimiter='\t'):
        if row['True pos'].strip() == '1':
            tps.append(row['Name'].strip())
        else:
            fps.append(row['Name'].strip())
    return tps, fps

