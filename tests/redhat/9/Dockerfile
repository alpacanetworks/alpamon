# Build: docker build -t alpamon:redhat-9 -f tests/redhat/9/Dockerfile .
# Run: docker run --mount type=bind,source="$(pwd)",target=/opt/alpamon alpamon:redhat-9

FROM redhat/ubi9

RUN yum install -y python3 python3-pip yum-utils
RUN curl -L https://pkg.osquery.io/rpm/GPG | tee /etc/pki/rpm-gpg/RPM-GPG-KEY-osquery && \
    yum-config-manager --add-repo https://pkg.osquery.io/rpm/osquery-s3-rpm.repo && \
    yum-config-manager --enable osquery-s3-rpm-repo && \
    yum install -y osquery

WORKDIR /opt/alpamon

COPY tests/redhat/9/alpamon.conf /etc/alpamon/alpamon.conf

RUN mkdir /var/log/alpamon
RUN --mount=type=bind,source=.,target=.,rw=true \
    /opt/alpamon/setup.py develop

CMD ["alpamon"]
