import os
import ssl
import logging


logger = logging.getLogger(__name__)

settings = {
    'SERVER_URL': None,
    'WS_PATH': '/ws/servers/backhaul/',
    'USE_SSL': False,
    'CA_CERT': None,
    'SSL_VERIFY': True,
    'SSL_OPT': {},
    'HTTP_THREADS': 4,
    'ID': None,
    'KEY': None,
}


def validate_config(config):
    logger.debug('Validating configuration fields...')
    valid = True
    val = config.get('server', 'url')
    if val.startswith('http://') or val.startswith('https://'):
        if val.endswith('/'):
            val = val[:-1]
        settings['SERVER_URL'] = val
        settings['WS_URL'] = val.replace('http', 'ws') + settings['WS_PATH']
        settings['USE_SSL'] = val.startswith('https://')
    else:
        logger.error('Server url is invalid.')
        valid = False

    if config.get('server', 'id') and config.get('server', 'key'):
        settings['ID'] = config.get('server', 'id')
        settings['KEY'] = config.get('server', 'key')
    else:
        logger.error("Server ID, KEY is empty")
        valid = False

    if settings['USE_SSL']:
        settings['SSL_VERIFY'] = config.getboolean('ssl', 'verify', fallback=True)
        ca_cert = config.get('ssl', 'ca_cert', fallback='')
        if not settings['SSL_VERIFY']:
            logger.warn(
                'SSL verification is turned off. '
                'Please be aware that this setting is not appropriate for production use.'
            )
            settings['SSL_OPT']['cert_reqs'] = ssl.CERT_NONE
        elif ca_cert:
            if not os.path.exists(ca_cert):
                logger.error('Given path for CA certificate does not exist.')
                valid = False
            else:
                settings['CA_CERT'] = ca_cert
                settings['SSL_OPT']['ca_certs'] = ca_cert

    return valid
