import json
import logging

from websocket import WebSocketApp

from alpamon.queryman import check_osquery
from alpamon.runner.command import CommandRunner
from alpamon.runner.commit import commit_async
from alpamon.io.queue import rqueue


logger = logging.getLogger(__name__)


class WebSocketClient(WebSocketApp):
    def __init__(self, api_session, ws_url, id, key):
        WebSocketApp.__init__(self, ws_url,
            on_open=WebSocketClient.on_open,
            on_message=WebSocketClient.on_message,
            on_error=WebSocketClient.on_error,
            on_close=WebSocketClient.on_close,
            header=['Authorization: id="%s", key="%s"' % (id, key)]
        )
        self.api_session = api_session
        self.running = True
        self.restart_requested = False
        self.closed = False

    def on_open(self):
        logger.debug('Websocket connection established.')

    def on_message(self, message):
        try:
            content = json.loads(message)
        except Exception as e:
            logger.exception(e)
            return
        
        if 'query' not in content:
            logger.error('Anappropriate message: %s' % message)
            return

        self.send_json({'query': 'hello'})

        try:
            # commit request handler
            if content['query'] == 'commit':
                # TODO: remove this handler in the future as it is deprecated
                logger.debug('Commit requested.')
                if check_osquery():
                    commit_async(self.api_session, content['commissioned'])
                else:
                    logger.error('Package "osquery" not found. Please install it first...')
                    self.quit()

            # command request handler
            elif content['query'] == 'command':
                command = content['command']
                rqueue.post(
                    '/api/events/commands/%(id)s/ack/' % command,
                    priority=10,
                )

                # execute command from the request
                # TODO: Handle commands that do not finish in certain period of time
                if content['command']['shell'] in ['internal', 'system', 'osquery']:
                    runner = CommandRunner(content['command'], self)
                    runner.start()
                else:
                    logger.error('Invalid command shell: %s.', content['command']['shell'])

            elif content['query'] == 'quit':
                logger.debug('Quit requested. reason: %s', content['reason'])
                self.quit()

            elif content['query'] == 'reconnect':
                logger.debug('Reconnect requested. reason: %s', content['reason'])
                self.close()

            else:
                logger.warn('Not implemented. query: %s', content['query'])

        except Exception as e:
            logger.exception(e)

    def on_error(self, error):
        if isinstance(error, (KeyboardInterrupt, SystemExit)):
            self.running = False
        else:
            logger.error(error)

    def on_close(self, close_status_code, close_msg):
        self.closed = True
        logger.debug('Websocket connection closed. %s', close_msg if close_msg != None else '')

    def send_json(self, json_data):
        self.send(json.dumps(json_data))

    def restart(self):
        self.restart_requested = True
        self.quit()

    def quit(self):
        self.running = False
        if not self.closed:
            self.close()
