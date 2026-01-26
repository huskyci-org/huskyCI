FROM eclipse-temurin:17-jdk-alpine

RUN apk update && apk upgrade \
	&& apk add --no-cache curl tar bash wget unzip openssh-client \
	&& apk add git

ARG SPOTBUGS_VERSION=4.9.8
ARG FINDSECBUGS_VERSION=1.14.0
ARG MAVEN_VERSION=3.9.12
ARG GRADLE_VERSION=9.3.0
ARG USER_HOME_DIR="/root"

RUN mkdir -p /usr/share/maven /usr/share/maven/ref \
  && curl -fsSL -o /tmp/apache-maven.tar.gz https://apache.osuosl.org/maven/maven-3/${MAVEN_VERSION}/binaries/apache-maven-${MAVEN_VERSION}-bin.tar.gz \
  && tar -xzf /tmp/apache-maven.tar.gz -C /usr/share/maven --strip-components=1 \
  && rm -f /tmp/apache-maven.tar.gz \
  && ln -s /usr/share/maven/bin/mvn /usr/bin/mvn \
  && mkdir -p /opt \
  && cd /opt \
  && wget -nc -O gradle.zip https://services.gradle.org/distributions/gradle-${GRADLE_VERSION}-bin.zip \
  && unzip gradle.zip \
  && rm -f gradle.zip \
  && mv gradle-${GRADLE_VERSION} gradle \
  && wget -nc -O spotbugs.zip https://repo.maven.apache.org/maven2/com/github/spotbugs/spotbugs/${SPOTBUGS_VERSION}/spotbugs-${SPOTBUGS_VERSION}.zip \
  && unzip spotbugs.zip \
  && rm -f spotbugs.zip \
  && mv spotbugs-${SPOTBUGS_VERSION} spotbugs \
  && wget -nc -O findsecbugs-plugin-${FINDSECBUGS_VERSION}.jar https://repo1.maven.org/maven2/com/h3xstream/findsecbugs/findsecbugs-plugin/${FINDSECBUGS_VERSION}/findsecbugs-plugin-${FINDSECBUGS_VERSION}.jar \
  && echo -n $SPOTBUGS_VERSION > /opt/spotbugs/version

ENV MAVEN_HOME /usr/share/maven
ENV MAVEN_CONFIG "$USER_HOME_DIR/.m2"
ENV GRADLE_USER_HOME "$USER_HOME_DIR/.gradle"

COPY mvn-entrypoint.sh /usr/local/bin/mvn-entrypoint.sh
COPY settings-docker.xml /usr/share/maven/ref/
COPY exclude.xml /opt/spotbugs/exclude.xml

VOLUME "$USER_HOME_DIR/.m2"
VOLUME "$USER_HOME_DIR/.gradle"
