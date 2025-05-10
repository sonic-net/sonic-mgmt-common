FROM sonicdev-microsoft.azurecr.io:443/sonic-slave-bookworm:latest 

RUN apt-get -y update
RUN apt-get install -y redis-server
RUN sed -ri 's/^# unixsocket/unixsocket/' /etc/redis/redis.conf
RUN sed -ri 's/^unixsocketperm .../unixsocketperm 777/' /etc/redis/redis.conf
RUN sed -ri 's/redis-server.sock/redis.sock/' /etc/redis/redis.conf

ARG LIBYANG_VER=1.0.73
COPY libyang*${LIBYANG_VER}*.deb /tmp/
RUN dpkg -i /tmp/libyang*${LIBYANG_VER}*.deb

COPY sonic_yang_models-1.0-py3-none-any.whl /tmp/
RUN pip3 install /tmp/sonic_yang_models-1.0-py3-none-any.whl
