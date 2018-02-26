FROM centos:centos7

# Provide access to host docker daemon
# TODO is this necessary within openshift?
# VOLUME ["/var/run/docker.sock"]

# Ports
EXPOSE 3004

# Run perceptor-scanner
CMD ["./perceptor-imagefacade"]
