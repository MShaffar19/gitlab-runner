FROM alpine:3.12.0

# gitlab-runner-helper will try to resolve `sh` from the path. We ensure the PATH is populated by default, as some container runtimes do no longer set a default (e.g. containerd v1.2.8)
ENV PATH="${PATH:-/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin}"

# hadolint ignore=DL3018
RUN apk add --no-cache bash ca-certificates git git-lfs miniperl \
    && ln -s miniperl /usr/bin/perl

RUN git lfs install --skip-repo

COPY ./scripts/ /usr/bin
COPY ./gitlab-runner-helper /usr/bin/

RUN echo 'hosts: files dns' >> /etc/nsswitch.conf

CMD ["sh"]
