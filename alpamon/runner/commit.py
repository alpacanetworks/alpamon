import logging


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
