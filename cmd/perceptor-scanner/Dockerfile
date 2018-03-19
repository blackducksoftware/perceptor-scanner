FROM centos as packager

RUN yum -y install wget tar

RUN mkdir /opt/java

#COPY jdk-9.0.4_linux-x64_bin.tar.gz /opt/java/

WORKDIR /opt/java

RUN wget --no-check-certificate -c --header "Cookie: oraclelicense=accept-securebackup-cookie" -q http://download.oracle.com/otn-pub/java/jdk/9.0.4+11/c2514751926b4512b076cc82f959763f/jdk-9.0.4_linux-x64_bin.tar.gz

RUN tar -xvf jdk-9.0.4_linux-x64_bin.tar.gz

RUN chmod 777 jdk-9.0.4/bin/jlink

# Build our own jdk9 distro with only these modules, as first stage
RUN jdk-9.0.4/bin/jlink \
    --module-path jdk-9.0.4/jmods \
    --verbose \
    --add-modules java.base,java.logging,java.xml,jdk.unsupported \
    --compress 2 \
    --no-header-files \
    --output /opt/jdk-9-minimal

FROM scratch

COPY --from=packager /opt/jdk-9-minimal /opt/jdk-9-minimal

ENV JAVA_HOME=/opt/jdk-9-minimal
ENV PATH="$PATH:$JAVA_HOME/bin"

# Bring in other dependencies
COPY ./perceptor-scanner ./perceptor-scanner

# Run perceptor-scanner
CMD ["./perceptor-scanner"]
