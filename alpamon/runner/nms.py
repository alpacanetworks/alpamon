import logging
import threading
import time
import requests
import datetime
from threading import Thread
from websocket import WebSocketApp

from alpamon.io.queue import rqueue
from alpamon.conf import settings


logger = logging.getLogger(__name__)


class SwitchLogReciever(threading.Thread):
    daemon = True

    def __init__(self):
        super().__init__()
        self.session = requests.Session()


class SNMPMessageReceiver(SwitchLogReciever):
    def __init__(self):
        super().__init__()
        self.name = 'SNMPMessageReceiver'

    def run(self):
        try:
            response = self.session.get(
                "http://localhost:5000/snmp/batch",
                stream=True,
            )
            response.raise_for_status()

            for line in response.iter_lines():
                if line:
                    rqueue.post(
                        '/api/nms/snmp/',
                        json=line.decode('utf-8'),
                        priority=80,
                    )
        except requests.exceptions.RequestException as e:
            raise Exception(e)


class SysLogReceiver(SwitchLogReciever):
    def __init__(self):
        super().__init__()
        self.name = 'SysLogReceiver'

    def run(self):
        try:
            response = self.session.get(
                "http://localhost:5000/syslog/batch",
                stream=True,
            )
            response.raise_for_status()

            for line in response.iter_lines():
                if line:
                    rqueue.post(
                        '/api/nms/syslog/',
                        json=line.decode('utf-8'),
                        priority=80,
                    )
        except requests.exceptions.RequestException as e:
            raise Exception(e)
        except Exception as e:
            raise Exception(e)


class LoggingClient(WebSocketApp):
    def __init__(self, session_id, log_type, url):
        WebSocketApp.__init__(self, url,
                              on_open=LoggingClient.on_open,
                              on_message=LoggingClient.on_message,
                              on_error=LoggingClient.on_error,
                              on_close=LoggingClient.on_close,
                              )
        self.session_id = session_id
        self.log_type = log_type
        self.session = requests.Session()

    def on_open(self):
        def read_stream(log_type):
            try:
                if log_type == 'snmp':
                    response = self.session.get(
                        "http://localhost:5000/snmp/stream",
                        stream=True,
                    )
                elif log_type == 'syslog':
                    response = self.session.get(
                        "http://localhost:5000/syslog/stream",
                        stream=True,
                    )

                response.raise_for_status()

                for line in response.iter_lines():
                    if line:
                        self.send(line.decode('utf-8'))
            except requests.exceptions.RequestException as e:
                raise Exception(e)

        t = threading.Thread(target=read_stream, args=self.log_type)
        t.start()

    def on_message(self, message):
        pass

    def on_error(self, error):
        if not self.closed:
            self.close()

    def on_close(self, close_status_code, close_msg):
        self.session.close()
        self.closed = True


def runlogging(session_id, log_type, url):
    client = LoggingClient(
        session_id=session_id,
        log_type=log_type,
        url=settings['SERVER_URL'].replace('http', 'ws') + url
    )
    client.run_forever(sslopt=settings['SSL_OPT'])


def call_settings_api(session, data):
    switch_id = data.pop('id')
    response = requests.get(
        'http://localhost:5000/settings',
        data=data
    )
    if response.status_code == 200:
        result = response.json()
        body = {
            'device': result['device'],
            'is_open': result['is_open'],
            'baud_rate': result['settings']['baudrate'],
            'byte_size': result['settings']['bytesize'],
            'parity': result['settings']['parity'],
            'stop_bits': result['settings']['stopbits'],
        }
        rqueue.patch(
            f'/api/nms/switches/{switch_id}/',
            json=body,
            priority=80,
        )
    else:
        raise Exception()


def call_commands_api(session, data):
    t_start = time.time()
    command_id = data.pop('id')
    rqueue.post(
        f'/api/nms/commands/{command_id}/ack/',
        priority=10,
    )
    response = requests.post(
        'http://localhost:5000/commands',
        data=data
    )
    if response.status_code == 200:
        result = response.json()
        t_end = time.time()
        body = {
            'success': result['exitcode'] == 0,
            'result': result['result'],
            'elapsed_time': (t_end - t_start),
        }
        rqueue.post(
            f'/api/nms/commands/{command_id}/fin/',
            json=body,
            priority=10,
        )
    else:
        raise Exception()


def call_scripts_api(session, data):
    t_start = time.time()
    script_id = data.pop('id')
    user_id = data.pop('requested_by')
    response = requests.post(
        'http://localhost:5000/scripts',
        data=data
    )
    if response.status_code == 200:
        result = response.json()
        t_end = time.time()
        body = {
            'script_id': script_id,
            'success': result['exitcode'] == 0,
            'result': result['result'],
            'elapsed_time': (t_end - t_start),
            'user_id': user_id
        }
        rqueue.post(
            '/api/nms/script-results/',
            json=body,
            priority=80,
        )
    else:
        raise Exception()


def is_thread_exist(name):
    thread_list = [thread.name for thread in threading.enumerate()]
    if name in thread_list:
        return True
    else:
        return False


def call_nms_async(session, data):
    if data['key'] == 'settings':
        Thread(
            target=call_settings_api,
            daemon=True,
            args=(session, data['body'])
        ).start()

        if not is_thread_exist('SNMPMessageReceiver'):
            snmp_receiver = SNMPMessageReceiver()
            snmp_receiver.start()

        if not is_thread_exist('SysLogReceiver'):
            syslog_receiver = SysLogReceiver()
            syslog_receiver.start()
    elif data['key'] == 'commands':
        Thread(
            target=call_commands_api,
            daemon=True,
            args=(session, data['body'])
        ).start()
    elif data['key'] == 'scripts':
        Thread(
            target=call_scripts_api,
            daemon=True,
            args=(session, data['body'])
        ).start()
    elif data['key'] == 'snmp/stream':
        t = threading.Thread(
            target=runlogging,
            name='SNMPLoggingThread',
            args=(data['session_id'], data['log_type'], data['url'])
        )
        t.daemon = True
        t.start()
    elif data['key'] == 'syslog/stream':
        t = threading.Thread(
            target=runlogging,
            name='SyslogLoggingThread',
            args=(data['session_id'], data['log_type'], data['url'])
        )
        t.daemon = True
        t.start()
    elif data['key'] == 'notification':
        pass
    else:
        logging.error('The %s API is not supported.' % data['key'])
