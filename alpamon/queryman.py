import json
import platform
import logging

from alpamon.runner.shell import runcmd

logger = logging.getLogger(__name__)


def get_osquery_path():
    system = platform.system().lower()
    if system == 'linux':
        osquery_path = '/usr/bin/osqueryi'
    elif system == 'darwin':
        osquery_path = '/usr/local/bin/osqueryi'
    elif system == 'windows':
        osquery_path = 'C:\ProgramData\osquery\osqueryi.exe'
    else:
        raise NotImplementedError('System %s not supported.' % system)
    return osquery_path


def check_osquery():
    (exitcode, output) = runcmd([get_osquery_path(),'--version'], include_stderr=False)
    if exitcode == 0:
        return True
    else:
        return False


def query(sql, output='json'):
    args = [get_osquery_path()]
    if output != 'table':
        args.append('--%s' % output)
    args.append(sql)
    (exitcode, result) = runcmd(args, include_stderr=False)
    if exitcode:
        return (exitcode, result)
    else:
        if result.startswith('Error:'):
            return (-1, result)
        if output == 'json':
            return (exitcode, json.loads(result))
        else:
            return (exitcode, result)
