FROM registry.fedoraproject.org/fedora
ENV HOME=/root
ENV PATH=$PATH:/root/fhir-benchmark
RUN set -euo pipefail; \
dnf upgrade -y; \
dnf install -y htop make go; \
mkdir -p /root/.config/htop; \
go mod init fhir-benchmark; \
go get gopkg.in/yaml.v3@latest;
COPY . $HOME/fhir-benchmark
WORKDIR $HOME/fhir-benchmark
RUN make all;
CMD /bin/bash