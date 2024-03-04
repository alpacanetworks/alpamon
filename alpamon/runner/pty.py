
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


logger = logging.getLogger(__name__)

terminals = {}


def get_env():
    return {
        'SHELL': '/bin/bash',
        'TERM': 'xterm-256color',
        'LS_COLORS': 'rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.Z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.jpg=01;35:*.jpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:',
        'LANG': 'en_US.UTF-8',
        'PATH': '%s:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin' % os.path.join(sys.exec_prefix, 'bin'),
    }


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
            env = get_env()
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
