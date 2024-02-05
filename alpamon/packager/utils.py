import logging
import platform

from alpamon.utils import platform_like
from alpamon.packager.system import SystemPackageManager


logger = logging.getLogger(__name__)


def get_system_package(session, name):
    arch = platform.machine()
    if arch == 'x86_64' and platform_like == 'debian':
        arch = 'amd64'
    elif arch == 'aarch64':
        arch = 'arm64'

    r = session.get('/api/packages/system/entries/', params={
        'package__name': name,
        'platform': platform_like,
        'arch': arch,
    })

    if r.status_code != 200:
        raise Exception('Server responded %d.' % r.status_code)

    packages = r.json()
    if not packages['results']:
        raise Exception('Package not found: (%s, %s, %s)' % (name, platform_like, arch))

    logger.debug('%d packages for "%s" are found.', packages['count'], name)
    package = packages['results'][0]

    logger.debug('Downloading %s...', package['name'])
    r = session.get(package['download_url'])
    if r.status_code != 200:
        raise Exception('Server responded %d.' % r.status_code)

    return (package['name'], r.content)


def get_python_package(session, name):
    r = session.get('/api/packages/python/entries/', params={
        'package__name': name,
        'target': 'py3',
    })
    if r.status_code != 200:
        raise Exception('Server responded %d.' % r.status_code)

    packages = r.json()
    if not packages['results']:
        raise Exception('Package not found: (%s, %s)' % (name, 'py3'))

    logger.debug('%d packages for "%s" are found.', packages['count'], name)
    package = packages['results'][0]

    logger.debug('Downloading %s...', package['name'])
    r = session.get(package['download_url'])
    if r.status_code != 200:
        raise Exception('Server responded %d.' % r.status_code)

    return (package['name'], r.content)


def install_osquery(session):
    (name, content) = get_system_package(session, 'osquery')

    logger.info('Installing %s...', name)

    (result, error) = SystemPackageManager.install_package_from_file(name, content)
    if result != 0:
        raise Exception(error)
