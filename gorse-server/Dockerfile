############################
# STEP 2 build a small image
############################
FROM scratch

ENV USER root

ADD ./server /usr/bin/gorse-server

ENTRYPOINT ["/usr/bin/gorse-server"]
