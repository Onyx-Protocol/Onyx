FROM codequest/ruby-golang-java-nodejs

RUN apt-get update
RUN apt-get install -y git

COPY startup.sh /startup.sh

ENV CHAIN /go/src/chain
ENV DOCS_DEST /generated
ENTRYPOINT /startup.sh
