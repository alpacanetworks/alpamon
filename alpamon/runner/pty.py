
import os
import pwd
import grp
import sys
import threading
import logging
import termios
import struct
import getpass
import signal
try:
    import fcntl
except:
    fcntl = None

from websocket import WebSocketApp

from alpamon.conf import settings
from alpamon.runner.env import get_default_env


logger = logging.getLogger(__name__)

terminals = {}


class PtyClient(WebSocketApp):
    def __init__(self, args, session_id, url, rows, cols, username, groupname, home_directory):
        WebSocketApp.__init__(self, url,
            on_open=PtyClient.on_open,
            on_message=PtyClient.on_message,
            on_error=PtyClient.on_error,
            on_close=PtyClient.on_close,
        )
        self.args = args
        self.session_id = session_id
        self.rows = rows
        self.cols = cols
        self.username = username
        self.groupname = groupname
        self.home_directory = home_directory

        self.fd = None
        self.pid = None
        self.closed = False

    def on_open(self):
        def read_tty():
            while True:
                try:
                    data = os.read(self.fd, 102400).decode('utf-8')
                except UnicodeDecodeError as e:
                    logger.debug('skipping %d bytes. %s', len(data), e)
                    continue
                except Exception as e:
                    logger.debug('failed to read data from the tty session.')
                    logger.exception(e)
                    break
                if not data:
                    logger.debug('tty session exited.')
                    break
                try:
                    self.send(data)
                except Exception as e:
                    logger.exception(e)
                    break
            
            os.close(self.fd)
            (_, status_code) = os.waitpid(self.pid, 0)
            logger.debug('exit status code: %d', status_code)
            self.pid = None
            if not self.closed:
                self.close()

        (pid, fd) = os.forkpty()
        if pid == 0:
            env = get_default_env()
            starting_uid = os.getuid()
            starting_gid = os.getgid()  
            starting_uid_name = pwd.getpwuid(starting_uid)[0]

            if starting_uid != 0:
                env['USER'] = getpass.getuser()
                env['HOME'] = os.path.expanduser('~/')
                print('Alpamon is not running as root. Falling back to "%s" account.' % env['USER'])
            else:
                env['USER'] = self.username
                env['HOME'] = self.home_directory

                # get uids
                user_uid = pwd.getpwnam(self.username)[2]
                user_gid = grp.getgrnam(self.groupname)[2]     
                gids = [g.gr_gid for g in grp.getgrall() if self.username in g.gr_mem and g.gr_name != "alpacon"]

                # drop privileges to given user
                os.setgid(user_gid)
                os.setgroups(gids)
                os.setuid(user_uid)
                
            # change directory and exec
            os.chdir(env['HOME'])
            os.execve(self.args[0], self.args, env)

        self.fd = fd
        self.pid = pid
        terminals[self.session_id] = self
        if fcntl != None and self.rows > 0 and self.cols > 0:
            try:
                self.resize(self.rows, self.cols, force=True)
            except Exception as e:
                logger.exception(e)

            t = threading.Thread(target=read_tty)
            t.daemon = True
            t.start()

    def on_message(self, message):
        try:
            os.write(self.fd, message.encode('utf-8'))
        except Exception as e:
            logger.exception(e)
            if not self.closed:
                self.close()

    def on_error(self, error):
        if not self.closed:
            self.close()

    def on_close(self, close_status_code, close_msg):
        self.closed = True
        if self.pid is not None:
            os.kill(self.pid, signal.SIGKILL)
        if self.session_id in terminals:
            del terminals[self.session_id]

    def resize(self, rows, cols, force=False):
        if not force and (rows == self.rows and cols == self.cols):
            logger.debug('Terminal size has not changed.')
            return

        size = struct.pack('HHHH', rows, cols, 0, 0)
        fcntl.ioctl(self.fd, termios.TIOCSWINSZ, size)
        self.rows = rows
        self.cols = cols
        logger.debug('Resized terminal for %(session_id)s to %(cols)dx%(rows)d.' % {
            'session_id': self.session_id,
            'cols': self.cols,
            'rows': self.rows,
        })


def runpty(args, session_id, url, rows, cols, username, groupname, home_directory):
    logger.debug('Opening websocket for pty session.')
    client = PtyClient(
        args, session_id, settings['SERVER_URL'].replace('http', 'ws') + url,
        rows, cols, username, groupname, home_directory
    )
    client.run_forever(sslopt=settings['SSL_OPT'])
    logger.debug('Websocket connection for pty has been closed.')


def runpty_bg(args, session_id, url, rows, cols, username, groupname, home_directory):
    t = threading.Thread(
        target=runpty,
        name='PtyThread',
        args=(args, session_id, url, rows, cols, username, groupname, home_directory),
    )
    t.daemon = True
    t.start()
