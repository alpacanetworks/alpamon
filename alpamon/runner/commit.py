import logging
from uuid import UUID
from threading import Thread, Lock

from alpamon import VERSION
from alpamon.queryman import query
from alpamon.utils import platform_like
from alpamon.io.queue import rqueue
from alpamon.packager.python import PythonPackageManager


logger = logging.getLogger(__name__)

lock = Lock()

COMMIT_DEFS = {
    'server': {
        'sql': {
            'osquery_version': 'SELECT `version` AS `osquery_version` FROM `osquery_info`',
            'load': 'SELECT `average` AS `load` FROM `load_average` WHERE `period`="1m"',
        },
        'multirow': False,
        'pk': 'version',
        'url': '/api/servers/servers/',
        'url_suffix': '-/sync/',
        'type': {
            'load': float
        }
    },
    'info': {
        'sql': 'SELECT `uuid`, `cpu_type`, `cpu_subtype`, `cpu_brand`, ''`cpu_physical_cores`, `cpu_logical_cores`, `physical_memory`, `hardware_vendor`, `hardware_model`, `hardware_version`, `hardware_serial`, `computer_name`, `hostname`, `local_hostname` FROM system_info',
        'multirow': False,
        'pk': 'uuid',
        'url': '/api/proc/info/',
        'url_suffix': '-/sync/',
        'type': {
            'cpu_logical_cores': int,
            'cpu_physical_cores': int,
            'physical_memory': int,
            'uuid': UUID,
        }
    },
    'os': {
        'sql': 'SELECT `name`, `version`, `major`, `minor`, `patch`, `build`, `platform`, `platform_like` FROM `os_version`',
        'multirow': False,
        'pk': 'name',
        'url': '/api/proc/os/',
        'url_suffix': '-/sync/',
        'type': {
            'major': int,
            'minor': int,
            'patch': int,
        }
    },
    'time': {
        'sql': 'SELECT `datetime`, `local_timezone` AS `timezone`, `total_seconds` AS `uptime` FROM `time` INNER JOIN `uptime`',
        'multirow': False,
        'pk': 'timezone',
        'url': '/api/proc/time/',
        'url_suffix': '-/sync/',
        'type': {
            'uptime': int,
        }
    },
    'groups': {
        'sql': 'SELECT `gid_signed` AS `gid`, `groupname` FROM groups',
        'multirow': True,
        'pk': 'gid',
        'url': '/api/proc/groups/',
        'url_suffix': 'sync/',
        'type': {
            'gid': int,
        }
    },
    'users': {
        'sql': 'SELECT `uid_signed` As `uid`, `gid_signed` AS `gid`, `username`, `description`, `directory`, `shell` FROM users',
        'multirow': True,
        'pk': 'uid',
        'url': '/api/proc/users/',
        'url_suffix': 'sync/',
        'type': {
            'gid': int,
            'uid': int,
        }
    },
    'interfaces': {
        'sql': 'SELECT interface AS name, mac, type, flags, mtu, link_speed FROM interface_details',
        'multirow': True,
        'pk': 'name',
        'url': '/api/proc/interfaces/',
        'url_suffix': 'sync/',
        'type': {
            'type': int,
            'flags': int,
            'mtu': int,
            'link_speed': int,
        }
    },
    'addresses': {
        'sql': 'SELECT `interface` AS `interface_name`, `address`, `mask`, `broadcast` FROM interface_addresses WHERE `address` NOT LIKE "fe80%"',
        'multirow': True,
        'pk': 'address',
        'url': '/api/proc/addresses/',
        'url_suffix': 'sync/',
        'type': {
        }
    },
    'packages': {
        'darwin': {
            'sql': 'SELECT name, path AS source, version FROM homebrew_packages',
            'multirow': True,
            'only': 'darwin',
            'pk': 'name',
            'url': '/api/proc/packages/',
            'url_suffix': 'sync/',
            'type': {
            }
        },
        'debian': {
            'sql': 'SELECT name, source, arch, version FROM deb_packages',
            'multirow': True,
            'only': 'debian',
            'pk': 'name',
            'url': '/api/proc/packages/',
            'url_suffix': 'sync/',
            'type': {
            }
        },
        'rhel': {
            'sql': 'SELECT name, source, arch, version FROM rpm_packages',
            'multirow': True,
            'only': 'rhel',
            'pk': 'name',
            'url': '/api/proc/packages/',
            'url_suffix': 'sync/',
            'type': {
            }
        }
    },
    'pypackages': {
        'multirow': True,
        'pk': 'name',
        'url': '/api/proc/pypackages/',
        'url_suffix': 'sync/',
        'type': {
        }
    }
}


def commit_system_info(session, keys=[]):
    data = {}

    if not keys:
        keys = list(COMMIT_DEFS.keys())

    for key in keys:
        if key not in list(COMMIT_DEFS.keys()):
            continue

        if key == 'packages':
            entry = COMMIT_DEFS[key][platform_like]
        else:
            entry = COMMIT_DEFS[key]

        if key == 'server':
            data['version'] = VERSION

            (exitcode, osquery_version) = query(entry['sql']['osquery_version'])
            data['osquery_version'] = osquery_version[0]['osquery_version'] if exitcode == 0 else None

            (exitcode, load) = query(entry['sql']['load'])
            data['load'] = load[0]['load'] if exitcode == 0 else None
        elif key == 'pypackages':
            data[key] = PythonPackageManager.list_packages()
        else:
            (exitcode, result) = query(entry['sql'])
            if exitcode == 0:
                data[key] = result if entry['multirow'] else result[0]
            else:
                logger.error('Failed to query information. sql: %s', entry['sql'])

    rqueue.put(
        '/api/servers/servers/-/commit/',
        json=data,
        priority=80,
    )
    rqueue.post(
        '/api/events/events/', json={
            'reporter': 'alpamon',
            'record': 'committed',
            'description': 'Committed system information. version: %(version)s' % {
                'version': VERSION,
            },
        },
        priority=80,
    )


def sync_system_info(session, keys=[]):
    with lock:
        if not keys:
            keys = list(COMMIT_DEFS.keys())

        for key in keys:
            if key == 'packages':
                entry = COMMIT_DEFS[key][platform_like]
            else:
                entry = COMMIT_DEFS[key]

            if key == 'server':
                (exitcode, osquery_result) = query(entry['sql']['osquery_version'])
                osquery_version = osquery_result[0]['osquery_version'] if exitcode == 0 else None
                (exitcode, load_result) = query(entry['sql']['load'])
                load = load_result[0]['load'] if exitcode == 0 else None
                data = {
                    'version': VERSION,
                    'osquery_version': osquery_version,
                    'load': load,
                }
                rqueue.patch(
                    entry['url'] + '-/sync/',
                    json=data,
                    priority=80,
                )
                continue

            elif key == 'pypackages':
                data = PythonPackageManager.list_packages()
            else:
                (exitcode, result) = query(entry['sql'])
                if exitcode == 0:
                    data = result
                else:
                    logger.error('Failed to query information. sql: %s', entry['sql'])

            for item in data:
                for k, func in entry['type'].items():
                    if func == UUID:
                        item[k] = str(func(item[k]))
                    else:
                        item[k] = func(item[k])

            if entry['multirow']:
                response = session.get(entry['url'] + entry['url_suffix'], timeout=10).json()
            else:
                response = [session.get(entry['url'] + entry['url_suffix'], timeout=10).json()]

            create_list, update_list, delete_dict = compare_data(key, entry, data, response)

            if create_list:
                rqueue.post(
                    entry['url'],
                    json=create_list,
                    priority=80,
                )

            for item in update_list:
                rqueue.patch(
                    entry['url'] + item[1]['id'] + '/',
                    json=item[0],
                    priority=80,
                )

            for item in delete_dict.values():
                rqueue.delete(
                    entry['url'] + item['id'] + '/',
                    json=item['data'],
                    priority=80,
                )


def compare_data(key, entry, data, response):
    response_dict = {}
    create_list = []
    compare_list = []
    update_list = []

    for item in response:
        if key == 'addresses' and not item['broadcast']:
            item['broadcast'] = ''

        if key == 'packages' and platform_like == 'darwin':
            try:
                del item['arch']
            except KeyError:
                pass

        uuid = item.pop('id')
        obj = {
            'id': uuid,
            'data': item
        }
        response_dict[item[entry['pk']]] = obj

    for item in data:
        if item[entry['pk']] in response_dict:
            compare_list.append((item, response_dict[item[entry['pk']]]))
            response_dict.pop(item[entry['pk']])
        else:
            create_list.append(item)

    for item in compare_list:
        if item[0] != item[1]['data']:
            update_list.append(item)
        else:
            pass

    if key in ['info', 'os', 'time']:
        create_list = []
        delete_dict = {}
    else:
        delete_dict = response_dict

    return create_list, update_list, delete_dict


def commit_async(session, commissioned):
    if commissioned:
        Thread(target=sync_system_info, daemon=True, args=(session,)).start()
    else:
        Thread(target=commit_system_info, daemon=True, args=(session,)).start()