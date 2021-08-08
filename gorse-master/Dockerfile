FROM scratch

ADD ./master /usr/bin/gorse-master

ADD ./config.toml /etc/config.toml

ENV USER root

ENTRYPOINT ["/usr/bin/gorse-master", "-c", "/etc/config.toml"]
