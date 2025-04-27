FROM alpine
LABEL maintainer="jonkey.guan@gmail.com"

COPY vfio-device-plugin /usr/bin/vfio-device-plugin

ENTRYPOINT ["/usr/bin/vfio-device-plugin", "-logtostderr=true", "-stderrthreshold=INFO", "-v=5"]
