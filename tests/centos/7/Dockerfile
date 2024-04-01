# Build: docker build -t alpamon:centos-7 -f tests/centos/7/Dockerfile .
# Run: docker run --mount type=bind,source="$(pwd)",target=/opt/alpamon alpamon:centos-7

FROM osquery/osquery:5.5.1-centos7

RUN yum install -y python3 python3-pip

WORKDIR /opt/alpamon

COPY tests/centos/7/alpamon.conf /etc/alpamon/alpamon.conf

RUN mkdir /var/log/alpamon
RUN --mount=type=bind,source=.,target=.,rw=true \
    /opt/alpamon/setup.py develop

CMD ["alpamon"]
