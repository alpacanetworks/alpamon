import os
import sys
import shutil

if sys.version_info >= (3, 9):
    import importlib.resources
else:
    from pkg_resources import resource_filename


DEFAULT_EDITOR = 'vi'

CONFIG_TARGET = '/etc/alpamon/%(name)s.conf'
TMPFILE_TARGET = '/usr/lib/tmpfiles.d/%(name)s.conf'
SERVICE_TARGET = '/lib/systemd/system/%(name)s.service'


def get_editor():
    return (os.environ.get('VISUAL') or os.environ.get('EDITOR') or DEFAULT_EDITOR)


def get_base_prefix_compat():
    """Get base/real prefix, or sys.prefix if there is none."""
    return (
        getattr(sys, 'base_prefix', None)
        or getattr(sys, 'real_prefix', None)
        or sys.prefix
    )


def in_virtualenv():
    return sys.prefix != get_base_prefix_compat()


def print_usage():
    sys.stderr.write('%s install|uninstall|configure\n' % sys.argv[0])
    sys.stderr.flush()


class ServiceManager:
    def __init__(self, name):
        self.name = name
        self.display_name = ' '.join(map(lambda x: x.capitalize(), self.name.split('-')))

        self.conf_file = CONFIG_TARGET % {'name': self.name }
        self.tmp_file = TMPFILE_TARGET % {'name': self.name }
        self.svc_file = SERVICE_TARGET % {'name': self.name }

    def get_resource_path(self, resource):
        if sys.version_info >= (3, 9):
            return importlib.resources.files(self.name.replace('-', '_')).joinpath(resource)
        else:
            return resource_filename(__name__, resource)

    def write_config(self):
        with self.get_resource_path('config/%s.conf' % self.name).open('r') as f:
            template = f.read()

        with open(self.conf_file, 'w') as f:
            f.write(template % {
                'url': os.environ.get('ALPACON_URL', 'https://alpacon.io'),
                'id': os.environ.get('PLUGIN_ID', ''),
                'key': os.environ.get('PLUGIN_KEY', ''),
                'verify': os.environ.get('ALPACON_SSL_VERIFY', 'true'),
                'ca_cert': os.environ.get('ALPACON_CA_CERT', ''),
                'debug': os.environ.get('PLUGIN_DEBUG', 'true'),
            })

    def write_service(self):
        if in_virtualenv():
            base_dir = sys.prefix
        else:
            base_dir = '/usr/local'
        exec_start = os.path.join(base_dir, 'bin', self.name)

        with self.get_resource_path('config/%s.service' % self.name).open('r') as f:
            template = f.read()

        with open(self.svc_file, 'w') as f:
            f.write(template % {
                'exec': exec_start,
                'display_name': self.display_name,
            })

    def configure(self):
        try:
            os.mkdir('/etc/alpamon')
            os.chmod('/etc/alpamon', 0o700)
        except FileExistsError:
            pass

        # copy configuration file if not exists
        if not os.path.exists(self.conf_file):
            self.write_config()

        # open an editor for the configuration file
        os.system('%s %s' % (get_editor(), self.conf_file))

    def install(self):
        print('Installing systemd service for %s...' % self.display_name)
        shutil.copyfile(
            self.get_resource_path('config/tmpfile.conf'),
            self.tmp_file
        )
        os.system('/bin/systemd-tmpfiles --create')

        self.write_config()
        self.write_service()

        os.system('/bin/systemctl daemon-reload')
        os.system('/bin/systemctl start %s.service' % self.name)
        os.system('/bin/systemctl enable %s.service' % self.name)
        os.system('/bin/systemctl --no-pager status %s.service' % self.name)
        print(
            '%s has been installed as a systemd service and '
            'will be launched automatically on system boot.' % self.display_name
        )

    def uninstall(self):
        print('Uninstalling systemd service for %s...' % self.display_name)
        os.system('/bin/systemctl stop %s.service' % self.name)
        os.system('/bin/systemctl disable %s.service' % self.name)
        os.remove(self.tmp_file)
        os.remove(self.svc_file)
        os.system('/bin/systemctl daemon-reload')
        print('Removing configuration files...')
        try:
            shutil.rmtree('/var/lib/alpamon')
        except:
            pass
        try:
            os.remove('/etc/alpamon/%s.conf' % self.name)
            os.rmdir('/etc/alpamon')
        except:
            pass
        print('%s has been removed successfully!' % self.display_name)
        print('Run "rm -rf /var/log/alpamon" to remove logs as well.')

    def run(self):
        if len(sys.argv) < 2:
            print_usage()
            sys.exit(1)

        if sys.argv[1] == 'install':
            self.install()
        elif sys.argv[1] == 'uninstall':
            self.uninstall()
        elif sys.argv[1] == 'configure':
            self.configure()
