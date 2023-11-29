import subprocess
import logging
import os
import pwd
import grp


logger = logging.getLogger(__name__)


def demote(username, groupname):
    # def result(): 
    logger.debug('starting demotion')
    logger.debug(username)
    logger.debug(groupname)

    if username is not None and groupname is not None:
        try:  
            user_uid = pwd.getpwnam(username)[2]
            user_gid = grp.getgrnam(groupname)[2]
            os.setgid(user_gid)
            os.setuid(user_uid)

        except Exception as e:
            logger.exception(e)
            raise Exception('There is no corresponding account in this server')

    logger.debug('finished demotion')


def runcmd(args, include_stderr=True, username=None, groupname=None):
    try:

        if (username == 'root'):            
            logger.debug('execute with root priv')
            result = subprocess.check_output(
                args,
                stderr=subprocess.STDOUT if include_stderr else subprocess.DEVNULL
            ).decode('utf-8')
        
        else:
            result = subprocess.check_output(
                args,
                preexec_fn=demote(username, groupname),
                stderr=subprocess.STDOUT if include_stderr else subprocess.DEVNULL
            ).decode('utf-8')

        return (0, result)
    except subprocess.CalledProcessError as e:
        logger.exception(e)
        return (e.returncode, '%s' % e.output.decode())
    except Exception as e:
        logger.exception(e)
        return (-1, '%s' % e)
