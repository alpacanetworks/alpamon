import time
from queue import PriorityQueue
from urllib3.util.retry import Retry

import requests
from requests.adapters import HTTPAdapter

from alpamon.logger.reporter import Reporter


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
