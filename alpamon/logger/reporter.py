import threading
import logging
import datetime


logger = logging.getLogger(__name__)


class Reporter(threading.Thread):
    name = 'Reporter'
    daemon = True

    def __init__(self, session, index=None):
        super().__init__()
        self.session = session
        if index != None:
            self.name = 'Reporter-%d' % index

    def report(self, entry):
        if entry.expiry and datetime.datetime.utcnow() > entry.expiry:
            return

        success = True
        try:
            r = self.session.request(
                entry.method,
                entry.url,
                json=entry.data,
                priority=entry.priority,
                buffered=False,
                timeout=10,
            )
            if int(r.status_code / 100) != 2:
                logger.debug(r.json())
                success = False
        except Exception as e:
            success = False

    def run(self):
        while True:
            try:
                entry = self.session.queue.get()
            except:
                continue
            self.report(entry)
