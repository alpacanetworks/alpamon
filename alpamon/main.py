import os
import time
import sys
import configparser
import logging
import logging.config
from threading import Thread
from http.client import responses

from pid import PidFile

from alpamon import VERSION
from alpamon.conf import settings, validate_config
from alpamon.client import WebSocketClient
from alpamon.runner.commit import commit_async
from alpamon.session import Session
from alpamon.queryman import check_osquery
from alpamon.packager.utils import install_osquery
from alpamon.logger.server import LogServer
from alpamon.logger.settings import LOGGING
from alpamon.logger.utils import configure


logger = logging.getLogger(__name__)


CONFIG_FILES = [
    '/etc/alpamon/alpamon.conf',
    os.path.expanduser('~/.alpamon.conf')
]

MIN_CONNECT_INTERVAL = 5
MAX_CONNECT_INTERVAL = 60


def init():
    configure('alpamon', LOGGING)

    config = configparser.ConfigParser()
    files = config.read(CONFIG_FILES)
    if len(files) == 0:
        print('Cannot find any configuration files.')
        print('Possible locations: %s' % ', '.join(CONFIG_FILES))
        sys.exit(1)
    print('Using config from %s.' % ', '.join(files))

    if config.getboolean('logging', 'debug') is True:
        LOGGING['loggers']['']['level'] = 'DEBUG'
        logging.config.dictConfig(LOGGING)

    if not validate_config(config):
        logger.error('Aborting...')
        sys.exit(1)

    return {
        'id': config.get('server', 'id'),
        'key': config.get('server', 'key'),
    }


def check_session(session):
    timeout = MIN_CONNECT_INTERVAL
    while True:
        try:
            r = session.get('/api/servers/servers/-/', timeout=5)
            if r.status_code in [200, 201]:
                return r.json()
            else:
                print('%s %s. %s' % (r.status_code, responses[r.status_code], r.text))
        except Exception as e:
            print(e)

        time.sleep(timeout)
        timeout *= 2
        if timeout > MAX_CONNECT_INTERVAL:
            timeout = MAX_CONNECT_INTERVAL


def main():
    with PidFile(pidname='alpamon') as p:
        creds = init()
        
        print('alpamon %s starting.' % VERSION)

        session = Session(settings, **creds)
        data = check_session(session)
        session.start_reporters()

        session.post('/api/events/events/', json={
            'reporter': 'alpamon',
            'record': 'started',
            'description': 'alpamon %(version)s started running.' % {
                'version': VERSION,
            },
        }, buffered=True)

        logserver = LogServer(session)
        
        if not check_osquery():
            try:
                install_osquery(session)
            except Exception as e:
                logger.exception(e)
                return

        commit_async(session, data['commissioned'])

        retry_interval = MIN_CONNECT_INTERVAL
        while True:
            logger.debug('Connecting %s...', settings['WS_URL'])
            client = WebSocketClient(session, ws_url=settings['WS_URL'], **creds)
            try:
                client.run_forever(sslopt=settings['SSL_OPT'])
                retry_interval = MIN_CONNECT_INTERVAL
            except Exception as e:
                retry_interval *= 2
                retry_interval = min(retry_interval, MAX_CONNECT_INTERVAL)
                logger.exception(e)

            if not client.running:
                break
            try:
                time.sleep(retry_interval)
            except KeyboardInterrupt:
                break

        logger.debug('Bye.')
        logserver.quit()

    if client.restart_requested:
        executable = sys.executable
        args = sys.argv[:]
        args.insert(0, sys.executable)
        os.execvp(executable, args)

    os._exit(0)


if __name__ == '__main__':
    main()
