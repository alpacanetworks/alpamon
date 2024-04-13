import time
import threading
import logging
import datetime

from alpamon.io.queue import rqueue


logger = logging.getLogger(__name__)


class Reporter(threading.Thread):
    name = 'Reporter'
    daemon = True

    def __init__(self, session, index=None):
        super().__init__()
        self.session = session
        if index != None:
            self.name = 'Reporter-%d' % index
        self.counters = {
            'success': 0,
            'failure': 0,
            'ignored': 0,
            'delay': 0,
            'latency': 0,
        }

    def query(self, entry):
        if entry.expiry and datetime.datetime.utcnow() > entry.expiry:
            self.counters['ignored'] += 1
            return

        success = True
        try:
            t1 = time.time()
            r = self.session.request(
                entry.method,
                entry.url,
                json=entry.data,
                priority=entry.priority,
                buffered=False,
                timeout=10,
            )
            t2 = time.time()
            self.counters['delay'] = self.counters['delay']*0.9 + (t2-entry.time)*0.1
            self.counters['latency'] = self.counters['latency']*0.9 + (t2-t1)*0.1
            if int(r.status_code / 100) != 2:
                logger.debug(r.json())
                success = False
        except Exception as e:
            success = False

        if success:
            self.counters['success'] += 1
        else:
            self.counters['failure'] += 1

    def run(self):
        while True:
            try:
                entry = rqueue.queue.get()
            except:
                continue
            self.query(entry)
