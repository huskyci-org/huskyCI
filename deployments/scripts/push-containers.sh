#!/bin/bash
#
# This script will push all securityTests containers
#

banditVersion=$(docker run --rm huskyciorg/bandit:latest bandit --version | grep bandit | awk -F " " '{print $2}')
brakemanVersion=$(docker run --rm huskyciorg/brakeman:latest brakeman --version | awk -F " " '{print $2}')
enryVersion=$(docker run --rm huskyciorg/enry:latest enry --version | cut -d'/' -f3)
gitAuthorsVersion=$(docker run --rm huskyciorg/gitauthors:latest git --version | awk -F " " '{print $3}')
gosecVersion=$(curl -s https://api.github.com/repos/securego/gosec/releases/latest | grep "tag_name" | awk -F '"' '{print $4}')
npmAuditVersion=$(docker run --rm huskyciorg/npmaudit:latest npm audit --version)
yarnAuditVersion=$(docker run --rm huskyciorg/yarnaudit:latest yarn audit --version )
safetyVersion=$(docker run --rm huskyciorg/safety:latest safety --version | awk -F " " '{print $3}')
gitleaksVersion=$(docker run --rm huskyciorg/gitleaks:latest gitleaks version)
spotbugsVersion=$(docker run --rm huskyciorg/spotbugs:latest cat /opt/spotbugs/version)
trivyVersion=$(docker run --rm huskyciorg/trivy:latest --version | awk -F " " '{print $2}')
securitycodescanVersion=$(docker run --rm huskyciorg/securitycodescan:latest security-scan | grep tool | awk -F " " '{print $6}')

docker tag "huskyciorg/bandit:latest" "huskyciorg/bandit:$banditVersion"
docker tag "huskyciorg/brakeman:latest" "huskyciorg/brakeman:$brakemanVersion"
docker tag "huskyciorg/enry:latest" "huskyciorg/enry:$enryVersion"
docker tag "huskyciorg/gitauthors:latest" "huskyciorg/gitauthors:$gitAuthorsVersion"
docker tag "huskyciorg/gosec:latest" "huskyciorg/gosec:$gosecVersion"
docker tag "huskyciorg/npmaudit:latest" "huskyciorg/npmaudit:$npmAuditVersion"
docker tag "huskyciorg/yarnaudit:latest" "huskyciorg/yarnaudit:$yarnAuditVersion"
docker tag "huskyciorg/safety:latest" "huskyciorg/safety:$safetyVersion"
docker tag "huskyciorg/gitleaks:latest" "huskyciorg/gitleaks:$gitleaksVersion"
docker tag "huskyciorg/spotbugs:latest" "huskyciorg/spotbugs:$spotbugsVersion"
docker tag "huskyciorg/trivy:latest" "huskyciorg/trivy:$trivyVersion"
docker tag "huskyciorg/securitycodescan:latest" "huskyciorg/securitycodescan:$securitycodescanVersion"

docker push "huskyciorg/bandit:latest" && docker push "huskyciorg/bandit:$banditVersion"
docker push "huskyciorg/brakeman:latest" && docker push "huskyciorg/brakeman:$brakemanVersion"
docker push "huskyciorg/enry:latest" && docker push "huskyciorg/enry:$enryVersion"
docker push "huskyciorg/gitauthors:latest" && docker push "huskyciorg/gitauthors:$gitAuthorsVersion"
docker push "huskyciorg/gosec:latest" && docker push "huskyciorg/gosec:$gosecVersion"
docker push "huskyciorg/npmaudit:latest" && docker push "huskyciorg/npmaudit:$npmAuditVersion"
docker push "huskyciorg/yarnaudit:latest" && docker push "huskyciorg/yarnaudit:$yarnAuditVersion"
docker push "huskyciorg/safety:latest" && docker push "huskyciorg/safety:$safetyVersion"
docker push "huskyciorg/gitleaks:latest" && docker push "huskyciorg/gitleaks:$gitleaksVersion"
docker push "huskyciorg/spotbugs:latest" && docker push "huskyciorg/spotbugs:$spotbugsVersion"
docker push "huskyciorg/trivy:latest" && docker push "huskyciorg/trivy:$trivyVersion"
docker push "huskyciorg/securitycodescan:latest" && docker push "huskyciorg/securitycodescan:$securitycodescanVersion"
