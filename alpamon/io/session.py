from urllib3.util.retry import Retry

import requests
from requests.adapters import HTTPAdapter

from alpamon.conf import settings


class Session(requests.Session):
    def __init__(self):
        super().__init__()
        self.base_url = settings['SERVER_URL']
        if settings['CA_CERT']:
            self.verify = settings['CA_CERT']
        if not settings['SSL_VERIFY']:
            self.verify = False
        self.headers.update({
            'Authorization': 'id="%s", key="%s"' % (settings['ID'], settings['KEY']),
        })

        adapter = HTTPAdapter(
            max_retries=Retry(total=1),
        )
        self.mount('http://', adapter)
        if settings['USE_SSL']:
            self.mount('https://', adapter)

    def request(self, method, url, json=None, **kwargs):
        if not url.startswith('http://') and not url.startswith('https://'):
            url = self.base_url + url
        return super().request(method, url, json=json, **kwargs)
