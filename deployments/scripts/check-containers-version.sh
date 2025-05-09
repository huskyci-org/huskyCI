#!/bin/bash
#
# This script will check the version of all securityTests
#


banditVersion=$(docker run --rm huskyciorg/bandit:latest bandit --version | grep bandit | awk -F " " '{print $2}')
brakemanVersion=$(docker run --rm huskyciorg/brakeman:latest brakeman --version | awk -F " " '{print $2}')
enryVersion=$(docker run --rm huskyciorg/enry:latest enry --version)
gitAuthorsVersion=$(docker run --rm huskyciorg/gitauthors:latest git --version | awk -F " " '{print $3}')
gosecVersion=$(docker run --rm huskyciorg/gosec:latest gosec --version | grep Version | awk -F " " '{print $2}')
npmAuditVersion=$(docker run --rm huskyciorg/npmaudit:latest npm audit --version)
yarnAuditVersion=$(docker run --rm huskyciorg/yarnaudit:latest yarn audit --version )
safetyVersion=$(docker run --rm huskyciorg/safety:latest safety --version | awk -F " " '{print $3}')
gitleaksVersion=$(docker run --rm huskyciorg/gitleaks:latest gitleaks --version)
spotbugsVersion=$(docker run --rm huskyciorg/spotbugs:latest cat /opt/spotbugs/version)
trivyVersion=$(docker run --rm huskyciorg/trivy:latest trivy --version | awk -F " " '{print $3}')
tfsecVersion=$(docker run --rm huskyciorg/tfsec:latest ./tfsec -v)
securitycodescanVersion=$(docker run --rm huskyciorg/securitycodescan:latest security-scan | grep tool | awk -F " " '{print $6}')

echo "bandit: $banditVersion"
echo "brakeman: $brakemanVersion"
echo "enry: $enryVersion"
echo "gitauthors: $gitAuthorsVersion"
echo "gosecVersion: $gosecVersion"
echo "npmauditVersion: $npmAuditVersion"
echo "yarnauditVersion: $yarnAuditVersion"
echo "safetyVersion: $safetyVersion"
echo "gitleaksVersion: $gitleaksVersion"
echo "spotbugsVersion: $spotbugsVersion"
echo "tfsecVersion: $tfsecVersion"
echo "securitycodescanVersion: $securitycodescanVersion"
