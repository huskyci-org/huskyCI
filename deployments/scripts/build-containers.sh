#!/bin/bash
#
# This script will build every securityTest container based on all dockerfiles from huskyCI repository
#

docker buildx build --platform linux/amd64 deployments/dockerfiles/bandit/ -t huskyciorg/bandit:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/brakeman/ -t huskyciorg/brakeman:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/enry/ -t huskyciorg/enry:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/gitauthors/ -t huskyciorg/gitauthors:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/gosec/ -t huskyciorg/gosec:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/npmaudit/ -t huskyciorg/npmaudit:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/npmaudit/ -t huskyciorg/yarnaudit:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/safety/ -t huskyciorg/safety:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/gitleaks/ -t huskyciorg/gitleaks:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/spotbugs/ -t huskyciorg/spotbugs:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/trivy/ -t huskyciorg/trivy:latest
docker buildx build --platform linux/amd64 deployments/dockerfiles/securitycodescan/ -t huskyciorg/securitycodescan:latest
