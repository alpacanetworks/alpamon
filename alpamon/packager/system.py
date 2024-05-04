import os
import logging

from alpamon.runner.shell import runcmd
from alpamon.utils import platform_like


logger = logging.getLogger(__name__)


def get_package_manager_cmd(request, source, name):
    if platform_like == 'debian':
        if request == 'install':
            if source == 'file':
                return ['dpkg', '--install', name]
            elif source == 'internet':
                return ['apt-get', 'install', '-y', ' --no-install-recommends', name]
        elif request == 'uninstall':
            if source == 'file':
                return ['dpkg', '--remove', name]
            elif source == 'internet':
                return ['apt-get', 'purge', '-y', name]
    
    elif platform_like == 'rhel':
        if request == 'install':
            if source == 'file':
                return ['rpm', '--install', name]
            elif source == 'internet':
                return ['yum', 'install', '-y', name]
        elif request == 'uninstall':
            if source == 'file':
                return ['rpm', '--erase', name]
            elif source == 'internet':
                return ['yum', 'erase', '-y', name]
    
    elif platform_like == 'darwin':
        if request == 'install':
            if source == 'file':
                return ['installer', '-pkg', name, '-target', '/']
            elif source == 'internet':
                return ['brew', 'install', name]
        elif request == 'uninstall':
            if source == 'internet':
                return ['brew', 'uninstall', name]
    
    # everything else
    raise NotImplementedError('Platform, request, or source not supported.')


class SystemPackageManager:
    @staticmethod
    def install_package(name):
        return runcmd(get_package_manager_cmd('install', 'internet', name))

    @staticmethod
    def install_package_from_file(name, data):
        try:
            with open(name, 'wb') as f:
                f.write(data)
        except Exception as e:
            logger.exception(e)
            if os.path.exists(name):
                os.remove(name)
            return (-1, 'Failed to write %s.' % name)
        
        # install the decoded wheel
        exitcode, result = runcmd(get_package_manager_cmd('install', 'file', name))
        if os.path.exists(name):
            os.remove(name)
        return (exitcode, result)

    @staticmethod
    def uninstall_package(name):
        return runcmd(get_package_manager_cmd('uninstall', 'internet', name))
