FROM ubuntu:22.04

RUN apt-get update && \
    DEBIAN_FRONTEND=noninteractive apt-get install -y \
    socat && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*


EXPOSE 12345

CMD ["/bin/sh", "-c", "socat TCP-LISTEN:12345,reuseaddr,fork EXEC:/bin/sh,pty,stderr,setsid,sigint,sane"]