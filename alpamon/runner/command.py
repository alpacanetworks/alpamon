import json
import time
import base64
import threading
import logging
import shlex
import traceback
import os
import pwd
import grp

from alpamon.queryman import query
from alpamon.runner.shell import runcmd
from alpamon.runner.pty import runpty_bg, terminals
from alpamon.packager.python import PythonPackageManager
from alpamon.packager.system import SystemPackageManager
from alpamon.packager.utils import get_python_package
from alpamon.utils import platform_like, now
from alpamon.runner.commit import commit_information


logger = logging.getLogger(__name__)


def deferred_runner(cmd, client):
    if cmd == 'restart':
        client.restart()
    elif cmd == 'quit':
        client.quit()


def get_file_data(session, data):
    content = None
    if 'type' not in data:
        raise ValueError('File type not specified.')

    if data['type'] == 'url':
        response = session.get(data['content'])
        content = response.content
    elif data['type'] == 'text':
        content = data['content'].encode()
    elif data['type'] == 'base64':
        content = base64.b64decode(data['content'])
    else:
        raise ValueError('Unknown file type: %s' % data['type'])
    if content is None:
        raise ValueError('Unknown content type.')
    return content


class CommandRunner(threading.Thread):
    name = 'CommandRunner'
    daemon = True

    def __init__(self, command, client):
        super().__init__()
        self.command = command
        self.client = client
        if 'id' in command and command['id'] != None:
            self.name = 'CommandRunner-%s' % command['id'].split('-')[-1]

    def commit(self, keys=[]):
        commit_information(self.client.api_session, keys=keys)

    @classmethod
    def commit_async(cls, client):
        CommandRunner({
            'id': None,
            'shell': 'internal',
            'line': 'commit',
        }, client).start()

    def handle_internal_cmd(self, command, data):
        args = shlex.split(command)

        # upgrade alpamon
        if args[0] == 'upgrade':
            (name, content) = get_python_package(self.client.api_session, 'alpamon')
            
            logger.info('Installing %s...', name)
            result = PythonPackageManager.install_package_from_wheel(name, content)
            self.commit(keys=['packages'])
            return result

        # commit
        elif args[0] == 'commit':
            self.commit(data.get('keys', []) if data else [])
            return (0, 'Committed system information.')

        # adduser
        elif args[0] == 'adduser':           
            data_fields = ['username', 'uid', 'gid', 'comment', 'home_directory', 'shell', 'groupname'] 
            
            # sanity check             
            if not all(data_field in data for data_field in data_fields): 
                raise Exception('Not enough information.')

            if platform_like == 'debian':
                '''adduser [--home DIR] [--shell SHELL] [--no-create-home] [--uid ID]
                    [--firstuid ID] [--lastuid ID] [--gecos GECOS] [--ingroup GROUP | --gid ID]
                    [--disabled-password] [--disabled-login] [--add_extra_groups]
                    [--encrypt-home] USER'''
                '''{"username": "eunyoung", "uid": 2000, "shell": "/bin/bash", "uid": 2001, "gid": 3001, "comment":"onlyfortest", "home_directory": "/home/eunyoung"}'''

                exitcode, result = runcmd([
                    '/usr/sbin/adduser',
                    '--home', data['home_directory'],
                    '--shell', data['shell'],
                    '--uid', str(data['uid']),
                    '--gid', str(data['gid']),
                    '--gecos', data['comment'],
                    '--disabled-password',
                    data['username']
                ])
                if exitcode != 0:
                    return (exitcode, result)
                
                for gid in data['groups']:
                    if gid == data['gid']:
                        continue

                    # get groupname from gid
                    groupname = grp.getgrgid(int(gid))[0]

                    # invoke adduser
                    exitcode, result = runcmd([
                        '/usr/sbin/adduser',
                        data['username'],
                        groupname
                    ])
                    if exitcode != 0:
                        return (exitcode, result)

            elif platform_like == 'rhel':
                exitcode, result = runcmd([
                    '/usr/sbin/useradd',
                    '--home-dir', data['home_directory'],
                    '--shell', data['shell'],
                    '--uid', str(data['uid']),
                    '--gid', str(data['gid']),
                    '--groups', ','.join(map(lambda x: str(x), data['groups'])),
                    '--comment', data['comment'],
                    data['username']
                ])
                if exitcode != 0:
                   return (exitcode, result)

            else:
                raise NotImplementedError()

            self.commit(keys=['groups', 'users'])
            return (0, 'Successfully added new user.')

        # addgroup
        elif args[0] == 'addgroup':
            data_fields = ['groupname', 'gid']

            # sanity check             
            if not all(data_field in data for data_field in data_fields): 
                raise Exception('Not enough information.')

            if platform_like == 'debian':
                '''addgroup [options] [--gid ID] group'''

                exitcode, result = runcmd([
                    '/usr/sbin/addgroup',
                    '--gid', str(data['gid']),
                    data['groupname'],
                ])
                if exitcode != 0:
                    return (exitcode, result)
            elif platform_like == 'rhel':
                exitcode, result = runcmd([
                    '/usr/sbin/groupadd',
                    '--gid', str(data['gid']),
                    data['groupname'],
                ])
                if exitcode != 0:
                   return (exitcode, result)

            else:
                raise NotImplementedError()

            self.commit(keys=['groups', 'users'])
            return (0, 'Successfully added new group.')

        # deluser
        elif args[0] == 'deluser':           
            data_fields = ['username']
            option_fields = ['remove-home', 'remove-all-files', '--backup']
            
            # sanity check             
            if not all(data_field in data for data_field in data_fields): 
                raise Exception('Not enough information.')

            if platform_like == 'debian':
                '''deluser [options] [--force] [--remove-home] [--remove-all-files] [--backup] [--backup-to DIR] user'''
                
                exitcode, result = runcmd([
                    '/usr/sbin/deluser',
                    data['username']
                ])
                if exitcode != 0:
                    return (exitcode, result)

            elif platform_like == 'rhel':
                exitcode, result = runcmd([
                    '/usr/sbin/userdel',
                    data['username'],
                ])
                if exitcode != 0:
                   return (exitcode, result)

            else:
                raise NotImplementedError()

            self.commit(keys=['groups', 'users'])
            return (0, 'Successfully deleted the user.')

        # delgroup
        elif args[0] == 'delgroup':           
            data_fields = ['groupname']
            
            # sanity check             
            if not all(data_field in data for data_field in data_fields): 
                raise Exception('Not enough information.')

            if platform_like == 'debian':
                '''delgroup [options] [--only-if-empty] group'''   

                exitcode, result = runcmd([
                    '/usr/sbin/delgroup',
                    data['groupname']
                ]) 
                if exitcode != 0:
                    return (exitcode, result)

            elif platform_like == 'rhel':
                exitcode, result = runcmd([
                    '/usr/sbin/groupdel',
                    data['groupname'],
                ])
                if exitcode != 0:
                   return (exitcode, result)

            else:
                raise NotImplementedError()

            self.commit(keys=['groups', 'users'])
            return (0, 'Successfully deleted the group.')

        # ping
        elif args[0] == 'ping':
            return (0, now())

        # open a pseudo terminal for shell
        elif args[0] == 'openpty':
            data_fields = ['session_id', 'url', 'username', 'groupname', 'home_directory', 'rows', 'cols'] 

            # sanity check
            if not all(data_field in data for data_field in data_fields):
                raise Exception('Not enough information.')

            runpty_bg(['/bin/bash', '-i'], **data)
            return (0, 'Spawned a pty terminal.')

        # resize pty terminal
        elif args[0] == 'resizepty':
            if data['session_id'] in terminals:
                terminals[data['session_id']].resize(data['rows'], data['cols'])
                return (0, 'Resized pty terminal to %(cols)dx%(rows)d.' % data)
            else:
                raise ValueError('Invalid session ID')

        # restart alpamon
        # defer running the command for a second to prevent race condition
        elif args[0] == 'restart':
            logger.info('Restart requested.')
            threading.Timer(1, deferred_runner, [args[0], self.client]).start()
            return (0, 'alpamon will restart in 1 second.')

        # quit alpamon
        # defer running the command for a second to prevent race condition
        elif args[0] == 'quit':
            logger.info('Quit requested.')
            threading.Timer(1, deferred_runner, [args[0], self.client]).start()
            return (0, 'alpamon will quit in 1 second.')
        
        # reboot system
        elif args[0] == 'reboot':
            logger.info('Reboot requested.')
            return self.handle_shell_cmd(
                'reboot',
                'root',
                'root'
            )

        # shutdown system
        elif args[0] == 'shutdown':
            logger.info('Shutdown requested.')
            return self.handle_shell_cmd(
                'shutdown',
                'root',
                'root'
            )

        # update system
        elif args[0] == 'update':
            logger.info('Upgrade system requested.')
            if platform_like == 'debian':
                line = 'apt-get update && apt-get upgrade -y && apt-get autoremove -y'
            elif platform_like == 'rhel':
                line = 'yum update -y'
            elif platform_like == 'darwin':
                line = 'brew upgrade'
            else:
                raise Exception('Platform "%s" not supported.' % platform_like)

            logger.debug('Running "%s"...', line)

            return self.handle_shell_cmd(
                line,
                'root',
                'root'
            )

        elif args[0] == 'help':
            return (0,
                'Available commands:\n\n'
                'upgrade: upgrade alpamon\n'
                'restart: restart alpamon\n'
                'quit: stop alpamon\n'
                'update: update system\n'
                'reboot: reboot system\n'
                'shutdown: shutdown systm\n'   
            )
        
        # invalid commands
        else:
            raise Exception('Invalid command: %s' % command)

    def handle_shell_cmd(self, command, user, group):
        spl = shlex.split(command)
        exitcode = 0
        results = ''

        if len(spl) > 0:
            logger.debug('Running "%s"', ' '.join(spl))
            exitcode, result = runcmd(spl, username=user, groupname=group)
            results += result
        return exitcode, results

    def run(self):
        t_start = time.time()
        if self.command['shell'] == 'internal':
            try:
                data = self.command.get('data', None)
                exitcode, result = self.handle_internal_cmd(
                    self.command['line'],
                    json.loads(data) if data else None
                )
            except:
                exitcode = 1
                result = traceback.format_exc()
        elif self.command['shell'] == 'system':
            exitcode, result = self.handle_shell_cmd(
                self.command['line'],
                self.command['user'],
                self.command['group']
            )
        elif self.command['shell'] == 'osquery':
            exitcode, result = query(self.command['line'], output='line')
        else:
            exitcode = 1
            result = 'Invalid command shell argument.'
        
        t_end = time.time()
        if result != None and self.command.get('id', None) != None:
            self.client.api_session.patch(
                '/api/events/commands/%(id)s/' % self.command, json={
                    'success': exitcode == 0,
                    'result': result,
                    'elapsed_time': (t_end-t_start),
                    'handled_at': now(),
                },
                priority=10,
                buffered=True,
            )

            # logger.debug('Sent response for command %s.', self.command['id'])
            # logger.debug('Result: \n%s', result)
