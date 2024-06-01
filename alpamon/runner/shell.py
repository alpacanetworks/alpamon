import subprocess
import logging
import os
import pwd
import grp
import sys

from alpamon.runner.env import get_default_env


logger = logging.getLogger(__name__)


def demote(username, groupname):
    def result():
        if username is not None and groupname is not None:
            if os.getuid() == 0:
                try:
                    user_uid = pwd.getpwnam(username)[2]
                    user_gid = grp.getgrnam(groupname)[2]
                    os.setgid(user_gid)
                    os.setuid(user_uid)

                    logger.debug('Demote permission to match %s, %s.', username, groupname)
                except Exception as e:
                    logger.exception(e)
                    raise Exception('There is no corresponding account in this server')
            else:
                logger.warn('Alpamon is not running as root. Falling back to the current user.')

    return result


def runcmd(args, include_stderr=True, username=None, groupname=None, env=None, timeout=3600):
    try:
        if env is not None:
            # set default environment variables
            default_env = get_default_env()
            for key in default_env:
                env.setdefault(key, default_env[key])

            # evaluate environment variables as they are not evaluated by `subprocess.check_output`
            for i in range(len(args)):
                if args[i].startswith('${') and args[i].endswith('}'):
                    var = env.get(args[i][2:-1], None)
                    if var is not None:
                        args[i] = var
                elif args[i].startswith('$'):
                    var = env.get(args[i][1:], None)
                    if var is not None:
                        args[i] = var

        if (username == 'root'):
            logger.debug('Executing the command with root privilege.')
            result = subprocess.check_output(
                args,
                stderr=subprocess.STDOUT if include_stderr else subprocess.DEVNULL,
                env=env,
                timeout=timeout,
            ).decode('utf-8')
        else:
            result = subprocess.check_output(
                args,
                preexec_fn=demote(username, groupname),
                stderr=subprocess.STDOUT if include_stderr else subprocess.DEVNULL,
                env=env,
                timeout=timeout,
            ).decode('utf-8')

        return (0, result)
    except subprocess.CalledProcessError as e:
        logger.exception(e)
        return (e.returncode, '%s' % e.output.decode())
    except Exception as e:
        logger.exception(e)
        return (-1, '%s' % e)
