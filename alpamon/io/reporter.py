import time
import threading
import logging
import datetime

from alpamon.conf import settings
from alpamon.io.queue import rqueue, RETRY_LIMIT
from alpamon.io.session import Session


logger = logging.getLogger(__name__)


class Reporter(threading.Thread):
    name = 'Reporter'
    daemon = True

    def __init__(self, index=None):
        super().__init__()
        if index != None:
            self.name = 'Reporter-%d' % index
        self.session = Session()
        self.counters = {
            'success': 0,
            'failure': 0,
            'ignored': 0,
            'delay': 0,
            'latency': 0,
        }

    def query(self, entry):
        try:
            t1 = time.time()
            r = self.session.request(
                entry.method,
                entry.url,
                json=entry.data,
            )
            t2 = time.time()
            self.counters['delay'] = self.counters['delay']*0.9 + (t2-entry.due)*0.1
            self.counters['latency'] = self.counters['latency']*0.9 + (t2-t1)*0.1

            if int(r.status_code / 100) == 2:
                success = True
            else:
                logger.debug(r.json())
                success = False
        except Exception as e:
            logger.error(entry.url)
            logger.exception(e)
            success = False

        if success:
            self.counters['success'] += 1
        else:
            self.counters['failure'] += 1
            if entry.retry > 0:
                entry.due += 2**(RETRY_LIMIT-entry.retry)  # exponantial backoff
                entry.retry -= 1
                rqueue.queue.put(entry)
            else:
                self.counters['ignored'] += 1

    def run(self):
        while True:
            try:
                entry = rqueue.queue.get()
            except:
                continue

            # ignore expired entries
            if entry.expiry and entry.expiry < datetime.datetime.utcnow():
                self.counters['ignored'] += 1

            # if the entry is not on due, schedule it again and sleep for a second
            elif entry.due and entry.due > time.time():
                try:
                    rqueue.queue.put(entry)
                except:
                    self.conuters['ignored'] += 1  # ignore if the queue is full
                time.sleep(1)

            # handle the entry
            else:
                self.query(entry)


reporters = []


def start_reporters():
    for i in range(settings['HTTP_THREADS']):
        reporter = Reporter(i)
        reporter.start()
        reporters.append(reporter)


def get_reporter_stats():
    return list(map(lambda x: x.counters, reporters))
