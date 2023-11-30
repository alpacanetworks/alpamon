import logging
import threading
import traceback
from concurrent import futures

import websocket
from websocket import WebSocketApp

from alpamon.conf import settings

logger = logging.getLogger(__name__)


class RproxyClient(WebSocketApp):
    def __init__(self, args, session_id, url, remote_url, port, headers):
        WebSocketApp.__init__(self, url,
                              on_open=RproxyClient.on_open,
                              on_message=RproxyClient.on_message,
                              on_error=RproxyClient.on_error,
                              on_close=RproxyClient.on_close,
                              )
        self.args = args
        self.session_id = session_id
        self.remote_url = 'ws://localhost:{}/{}'.format(port, remote_url)
        # self.remote_url = 'ws://localhost:{}'.format(port)
        self.port = port
        self.remote_ws = None

        self.closed = False

    def on_open(self):
        logger.debug('Rproxy Websocket connection established.')

    def on_message(self, message):
        self.remote_ws.send(message, websocket.ABNF.OPCODE_BINARY)

    def on_error(self, error):
        if not self.closed:
            self.close()

    def on_close(self, close_status_code, close_msg):
        self.closed = True
        logger.debug('Rproxy Websocket connection closed. %s', close_msg if close_msg != None else '')


class RemoteWSClient(WebSocketApp):
    def __init__(self, url):
        WebSocketApp.__init__(self, url,
                              on_open=RemoteWSClient.on_open,
                              on_message=RemoteWSClient.on_message,
                              on_error=RemoteWSClient.on_error,
                              on_close=RemoteWSClient.on_close,
                              )

        self.remote_ws = None
        self.closed = False

    def on_open(self):
        logger.debug('Remote Websocket connection established.')

    def on_message(self, message):
        self.remote_ws.send(message, websocket.ABNF.OPCODE_BINARY)

    def on_error(self, error):
        if not self.closed:
            self.close()

    def on_close(self, close_status_code, close_msg):
        self.closed = True
        logger.debug('Remote Websocket connection closed. %s', close_msg if close_msg != None else '')


def get_rproxy_client(args, session_id, url, remote_url, port, headers):
    return RproxyClient(
        args, session_id, settings['SERVER_URL'].replace('http', 'ws') + url, remote_url, port, headers
    )


def get_remote_client(remote_url, port):
    url = 'ws://localhost:{}/{}'.format(port, remote_url)

    return RemoteWSClient(url)


def runrproxy(ws, remote_ws):
    logger.debug('Opening websocket for rproxy session.')
    ws.remote_ws = remote_ws
    ws.run_forever(sslopt=settings['SSL_OPT'])

    logger.debug('Websocket connection for rproxy has been closed.')


def runremote(ws, remote_ws):
    logger.debug('Opening websocket for remote session.')
    ws.remote_ws = remote_ws
    ws.run_forever(sslopt=settings['SSL_OPT'])

    logger.debug('Websocket connection for remote has been closed.')


def runrproxy_bg(args, session_id, url, remote_url, port, headers):
    # websocket.enableTrace(True)
    rproxy_client = get_rproxy_client(args, session_id, url, remote_url, port, headers)
    remote_client = get_remote_client(remote_url, port)

    rproxy_thread = threading.Thread(
        target=runrproxy,
        name='RproxyThread',
        args=(rproxy_client, remote_client),
    )
    remote_thread = threading.Thread(
        target=runremote,
        name='RemoteThread',
        args=(remote_client, rproxy_client),
    )

    rproxy_thread.daemon = True
    remote_thread.daemon = True

    rproxy_thread.start()
    remote_thread.start()
