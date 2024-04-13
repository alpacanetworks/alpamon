import time
from queue import PriorityQueue


RETRY_LIMIT = 5


class PriorityEntry:
    def __init__(self, priority, method, url, data, due=None, expiry=None):
        self.priority = priority
        self.method = method
        self.url = url
        self.data = data
        self.due = time.time() if due is None else due  # entries will not be handled until due
        self.expiry = expiry      # expired entries are ignored
        self.retry = RETRY_LIMIT  # remaning retry count

    def __lt__(self, other):
        if self.priority == other.priority:
            return self.due < other.due
        else:
            return self.priority < other.priority

    def __str__(self):
        return '(%d, %s, %s)' % (self.priority, self.due, self.url)


class RequestQueue:
    def __init__(self):
        self.queue = PriorityQueue(maxsize=10*60*60)  # 10 entries/second * 1h
    
    def request(self, method, url, json=None, priority=10, due=None, expiry=None, **kwargs):
        self.queue.put(PriorityEntry(priority, method, url, json, due, expiry))

    def post(self, url, json=None, priority=10, **kwargs):
        return self.request('POST', url, json=json, priority=priority, **kwargs)

    def patch(self, url, json=None, priority=10, **kwargs):
        return self.request('PATCH', url, json=json, priority=priority, **kwargs)

    def put(self, url, json=None, priority=10, **kwargs):
        return self.request('PUT', url, json=json, priority=priority, **kwargs)


rqueue = RequestQueue()
