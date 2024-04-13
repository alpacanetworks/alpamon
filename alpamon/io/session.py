import time
from queue import PriorityQueue
from urllib3.util.retry import Retry

import requests
from requests.adapters import HTTPAdapter

from alpamon.io.reporter import Reporter


class PriorityEntry:
    def __init__(self, priority, method, url, data, expiry=None):
        self.priority = priority
        self.time = time.time()
        self.method = method
        self.url = url
        self.data = data
        self.expiry = expiry

    def __lt__(self, other):
        if self.priority == other.priority:
            return self.time < other.time
        else:
            return self.priority < other.priority

    def __str__(self):
        return '(%d, %s, %s)' % (self.priority, self.time, self.url)


class Session(requests.Session):
    def __init__(self, settings, id, key):
        super().__init__()
        self.settings = settings
        
        self.queue = PriorityQueue(maxsize=10*60*60)  # 10 entries/second * 1h
        self.base_url = self.settings['SERVER_URL']
        if self.settings['CA_CERT']:
            self.verify = self.settings['CA_CERT']
        if not self.settings['SSL_VERIFY']:
            self.verify = False
        self.headers.update({
            'Authorization': 'id="%s", key="%s"' % (id, key),
        })
        self.reporters = []
        self.num_threads = self.settings['HTTP_THREADS']

    def start_reporters(self):
        adapter = HTTPAdapter(
            pool_connections=self.num_threads,
            pool_maxsize=self.num_threads,
            max_retries=Retry(total=1),
        )
        self.mount('http://', adapter)
        if self.settings['USE_SSL']:
            self.mount('https://', adapter)

        for i in range(self.num_threads):
            reporter = Reporter(self, i)
            reporter.start()
            self.reporters.append(reporter)

    def get_reporter_stats(self):
        return list(map(lambda x: x.counters, self.reporters))

    def request(self, method, url, json=None, priority=10, buffered=False, expiry=None, **kwargs):
        if buffered:
            self.queue.put(PriorityEntry(priority, method, url, json, expiry))
            return None
        else:
            if not url.startswith('http://') and not url.startswith('https://'):
                url = self.base_url + url
            return super().request(method, url, json=json, **kwargs)

    def get(self, url, **kwargs):
        return super().get(url, **kwargs)

    def post(self, url, json=None, priority=10, buffered=False, **kwargs):
        return self.request('POST', url, json=json, priority=priority, buffered=buffered, **kwargs)

    def patch(self, url, json=None, priority=10, buffered=False, **kwargs):
        return self.request('PATCH', url, json=json, priority=priority, buffered=buffered, **kwargs)

    def put(self, url, json=None, priority=10, buffered=False, **kwargs):
        return self.request('PUT', url, json=json, priority=priority, buffered=buffered, **kwargs)
