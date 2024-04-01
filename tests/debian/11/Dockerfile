# Build: docker build -t alpamon:debian-11 -f tests/debian/11/Dockerfile .
# Run: docker run --mount type=bind,source="$(pwd)",target=/opt/alpamon alpamon:debian-11

FROM debian:11

ENV OSQUERY_KEY=1484120AC4E9F8A1A577AEEE97A80C63C9D8B80B

ARG DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends python3 python3-pip python3-setuptools gnupg software-properties-common apt-transport-https
RUN apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys $OSQUERY_KEY && \
    add-apt-repository 'deb https://pkg.osquery.io/deb deb main' && \
    apt-get update && apt-get install -y --no-install-recommends osquery

WORKDIR /opt/alpamon

COPY tests/debian/11/alpamon.conf /etc/alpamon/alpamon.conf

RUN mkdir /var/log/alpamon
RUN --mount=type=bind,source=.,target=.,rw=true \
    /opt/alpamon/setup.py develop

CMD ["alpamon"]
