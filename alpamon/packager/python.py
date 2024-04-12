import os
import sys
import platform
import logging
import json

from alpamon.runner.shell import runcmd


logger = logging.getLogger(__name__)

PIP_PATH = os.path.join(sys.prefix, 'bin', 'pip3')


class PythonPackageManager:
    @staticmethod
    def pyversion():
        (exitcode, result) = runcmd(['python2', '-V'])
        return {
            'python2': result.split()[1] if exitcode == 0 else None,
            'python3': platform.python_version()
        }

    @staticmethod
    def list_packages():
        (exitcode, result) = runcmd(
            [PIP_PATH, 'list', '--format', 'json', '--disable-pip-version-check'],
            include_stderr=False
        )
        if exitcode != 0 or result.startswith('Error:'):
            return None
        try:
            packages = []
            for package in json.loads(result):
                item = {
                    'name': package['name'],
                    'version': package['version']
                }
                packages.append(item)
            return packages
        except Exception as e:
            logger.debug(result)
            logger.exception(e)
            return None

    @staticmethod
    def install_package_from_pip(name, version=None):
        if version is not None:
            package = '%s==%s' % (name, version)
        else:
            package = name
        return runcmd([PIP_PATH, 'install', '-U', package])

    @staticmethod
    def install_package_from_wheel(name, data):
        # write a .whl file
        try:
            with open(name, 'wb') as f:
                f.write(data)
        except Exception as e:
            logger.exception(e)
            if os.path.exists(name):
                os.remove(name)
            return (-1, 'Failed to write %s.' % name)
        
        # install the decoded wheel
        exitcode, result = runcmd([PIP_PATH, 'install', '-U', name])
        if os.path.exists(name):
            os.remove(name)
        return (exitcode, result)

    @staticmethod
    def uninstall_package(name):
        return runcmd([PIP_PATH, 'uninstall', '-y', name])
