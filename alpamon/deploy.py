import sys

from alpamon.service import Service, print_usage


def main():
    if len(sys.argv) < 2:
        print_usage()
        sys.exit(1)

    service = Service('alpamon')
    if sys.argv[1] == 'install':
        service.install()
    elif sys.argv[1] == 'uninstall':
        service.uninstall()
    elif sys.argv[1] == 'configure':
        service.configure()


if __name__ == '__main__':
    main()
