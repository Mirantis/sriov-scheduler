FROM alpine:3.6
LABEL Name="sriov-scheduler-extender" Version="0.1"
COPY discovery /usr/sbin/
COPY extender /usr/sbin
ENTRYPOINT ["/usr/sbin/extender"]
