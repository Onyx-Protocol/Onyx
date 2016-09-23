FROM golang:1.7

# clang-format
RUN \
	( curl -sL https://deb.nodesource.com/setup_6.x | bash - ) \
	&& apt-get update \
	&& apt-get install -y --no-install-recommends \
		clang-format-3.5 \
		nodejs \
	&& rm -rf /var/lib/apt/lists/* \
	&& go get \
		golang.org/x/tools/cmd/goimports
