FROM centos:centos7


# Get Java
RUN yum update -y && yum install -y -q java-1.8.0-openjdk && yum clean all
#RUN BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/
ENV BDS_JAVA_HOME=/usr/lib/jvm/java-1.8.0-openjdk-1.8.0.151-5.b12.el7_4.x86_64/jre/


# Bring in other dependencies
COPY ./perceptor-scanner ./perceptor-scanner


# Run perceptor-scanner
CMD ["./perceptor-scanner"]
