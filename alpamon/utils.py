import platform
import datetime

import distro


def get_platform_like():
    system = platform.system().lower()
    if system == 'linux':
        result = distro.id()
        if result in ['ubuntu', 'debian']:
            result = 'debian'
        elif result in ['centos', 'rhel']:
            result = 'rhel'
        else:
            raise NotImplementedError('Platform %s not supported.' % result)
    elif system == 'windows':
        result = 'windows'
    elif system == 'darwin':
        result = 'darwin'
    else:
        raise NotImplementedError('Platform %s not supported.' % platform.system())
    return result


def now():
    return datetime.datetime.now(datetime.timezone.utc).astimezone().isoformat()


platform_like = get_platform_like()
