import logging

from alpamon import VERSION
from alpamon.queryman import query
from alpamon.utils import platform_like
from alpamon.packager.python import PythonPackageManager


logger = logging.getLogger(__name__)


COMMIT_DEFS = [
    {
        'key': 'info',
        'sql': 'SELECT `uuid`, `cpu_type`, `cpu_subtype`, `cpu_brand`, ''`cpu_physical_cores`, `cpu_logical_cores`, `physical_memory`, `hardware_vendor`, `hardware_model`, `hardware_version`, `hardware_serial`, `computer_name`, `hostname`, `local_hostname` FROM system_info',
        'multirow': False,
    }, {
        'key': 'os',
        'sql': 'SELECT `name`, `version`, `major`, `minor`, `patch`, `build`, `platform`, `platform_like` FROM `os_version`',
        'multirow': False,
    }, {
        'key': 'time',
        'sql': 'SELECT `datetime`, `local_timezone` AS `timezone`, `total_seconds` AS `uptime` FROM `time` INNER JOIN `uptime`',
        'multirow': False,
    }, {
        'key': 'load',
        'sql': 'SELECT `period`, `average` FROM `load_average` WHERE `period`="1m"',
        'multirow': False,
    }, {
        'key': 'groups',
        'sql': 'SELECT `gid_signed` AS `gid`, `groupname` FROM groups',
        'multirow': True,
    }, {
        'key': 'users',
        'sql': 'SELECT `uid_signed` As `uid`, `gid_signed` AS `gid`, `username`, `description`, `directory`, `shell` FROM users',
        'multirow': True,
    }, {
        'key': 'interfaces',
        'sql': 'SELECT interface AS name, mac, type, flags, mtu, link_speed FROM interface_details',
        'multirow': True,
    },{
        'key': 'addresses',
        'sql': 'SELECT `interface` AS `interface_name`, `address`, `mask`, `broadcast` FROM interface_addresses WHERE `address` NOT LIKE "fe80%"',
        'multirow': True,
    }, {
        'key': 'packages',
        'sql': 'SELECT name, path AS source, version FROM homebrew_packages',
        'multirow': True,
        'only': 'darwin',
    }, {
        'key': 'packages',
        'sql': 'SELECT name, source, arch, version FROM deb_packages',
        'multirow': True,
        'only': 'debian',
    }, {
        'key': 'packages',
        'sql': 'SELECT name, source, arch, version FROM rpm_packages',
        'multirow': True,
        'only': 'rhel',
    }
]


def commit_information(session, keys=[]):
    data = {
        'version': VERSION,
    }

    if not keys or 'osquery_version' in keys:
        (exitcode, result) = query('SELECT `version` FROM `osquery_info`')
        data['osquery_version'] = result[0]['version'] if exitcode == 0 else None

    if not keys or 'pypackages' in keys:
        data['pypackages'] = PythonPackageManager.list_packages()

    for entry in COMMIT_DEFS:
        if keys and not entry['key'] in keys:
            continue
        if 'only' in entry and entry['only'] != platform_like:
            continue
        (exitcode, result) = query(entry['sql'])
        if exitcode == 0:
            data[entry['key']] = result if entry['multirow'] else result[0]
        else:
            logger.error('Failed to query information. sql: %s', entry['sql'])

    session.put(
        '/api/servers/servers/-/commit/',
        json=data,
        priority=80,
        buffered=True,
    )
    session.post(
        '/api/events/events/', json={
            'reporter': 'alpamon',
            'record': 'committed',
            'description': 'Committed system information. version: %(version)s' % {
                'version': VERSION,
            },
        }, priority=80, buffered=True,
    )
