from urllib3.util.retry import Retry

import requests
from requests.adapters import HTTPAdapter


class Session(requests.Session):
    def __init__(self, settings, id, key):
        super().__init__()
        self.settings = settings

        self.base_url = self.settings['SERVER_URL']
        if self.settings['CA_CERT']:
            self.verify = self.settings['CA_CERT']
        if not self.settings['SSL_VERIFY']:
            self.verify = False
        self.headers.update({
            'Authorization': 'id="%s", key="%s"' % (id, key),
        })

        adapter = HTTPAdapter(
            max_retries=Retry(total=3),
        )
        self.mount('http://', adapter)
        if self.settings['USE_SSL']:
            self.mount('https://', adapter)

    def request(self, method, url, json=None, **kwargs):
        if not url.startswith('http://') and not url.startswith('https://'):
            url = self.base_url + url
        return super().request(method, url, json=json, **kwargs)
