import pickle
import logging
import logging.handlers
import threading
import socketserver
import struct
import datetime

from alpamon.io.queue import rqueue


logger = logging.getLogger(__name__)


class LogRecordStreamHandler(socketserver.StreamRequestHandler):
    def handle(self):
        while True:
            chunk = self.connection.recv(4)
            if len(chunk) < 4:
                break
            slen = struct.unpack('>L', chunk)[0]
            chunk = self.connection.recv(slen)
            while len(chunk) < slen:
                chunk = chunk + self.connection.recv(slen - len(chunk))
            obj = pickle.loads(chunk)
            record = logging.makeLogRecord(obj)
            self.handle_record(record)

    def handle_record(self, record):
        date = datetime.datetime.utcfromtimestamp(record.created)
        rqueue.post(
            '/api/history/logs/',
            json={
                'date': '%sZ' % date.isoformat(),
                'level': record.levelno,
                'program': record.program,
                'name': record.name,
                'path': record.pathname,
                'lineno': record.lineno,
                'pid': record.process,
                'tid': record.thread,
                'process': record.processName,
                'thread': record.threadName,
                'msg': record.msg,
            },
            priority=90,
        )


class LogRecordSocketReceiver(socketserver.ThreadingTCPServer):
    allow_reuse_address = True
    daemon_threads = True

    def __init__(self, host='localhost',
            port=logging.handlers.DEFAULT_TCP_LOGGING_PORT,
            handler=LogRecordStreamHandler):
        socketserver.ThreadingTCPServer.__init__(self, (host, port), handler)


class LogServer:
    def __init__(self):
        self.server = LogRecordSocketReceiver()
        self.thread = threading.Thread(
            target=self.server.serve_forever,
            name='LogServer',
        )
        self.thread.daemon = True
        self.thread.start()
        logger.debug(
            'Started log server on localhost:%d.',
            logging.handlers.DEFAULT_TCP_LOGGING_PORT
        )

    def quit(self):
        self.server.shutdown()
        self.server.server_close()
        self.thread.join()
        logger.debug('Stopped log server.')
