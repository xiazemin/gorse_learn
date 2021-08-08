############################
# STEP 2 build a small image
############################
FROM scratch

ADD ./worker /usr/bin/gorse-worker

ENV USER root

ENTRYPOINT ["/usr/bin/gorse-worker"]
