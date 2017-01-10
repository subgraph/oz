FROM fedora

RUN dnf -y install go
RUN dnf -y install git
RUN dnf -y install make

ENV GOPATH /root/gopath
ENV PATH=$PATH:$GOPATH/bin

RUN mkdir -p /root/gopath/src/github.com/twtiger/gosecco
