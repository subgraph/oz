FROM fedora:25

RUN dnf -y install \
  git \
  go \
  make \
&& dnf clean all

ENV GOPATH /root/gopath
ENV PATH=$PATH:$GOPATH/bin

COPY . "$GOPATH/src/github.com/twtiger/gosecco"
WORKDIR "$GOPATH/src/github.com/twtiger/gosecco"

RUN make deps-dev

CMD ["/bin/bash"]
