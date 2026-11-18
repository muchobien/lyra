FROM debian:trixie-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates curl git gcc libc6-dev jq less netcat-openbsd openssh-server procps ripgrep unzip xz-utils \
    && rm -rf /var/lib/apt/lists/* /etc/ssh/ssh_host_* \
    && useradd --create-home --uid 1000 --shell /bin/bash dev \
    && mkdir -p /opt/mise \
    && chown dev:dev /opt/mise

ENV MISE_DATA_DIR=/opt/mise \
    MISE_CACHE_DIR=/opt/mise/cache \
    MISE_YES=1 \
    PATH=/opt/mise/shims:/opt/mise/bin:/home/dev/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN curl -fsSL https://mise.run | MISE_INSTALL_PATH=/usr/local/bin/mise sh

COPY gitconfig /etc/gitconfig
COPY sshd_config /etc/ssh/sshd_config.d/orca.conf
COPY profile.sh /etc/profile.d/orca.sh
COPY --chmod=0755 entrypoint.sh /usr/local/bin/orca-workspace-entrypoint

USER dev
WORKDIR /home/dev

COPY --chown=dev:dev repo/ /tmp/repo/
RUN curl -fsSL https://claude.ai/install.sh | bash \
    && mise use --global node@26 gh@latest npm:@openai/codex@latest \
    && cd /tmp/repo \
    && mise trust \
    && mise install \
    && mise exec -- go mod download \
    && cd web \
    && mise exec -- pnpm fetch \
    && rm -rf /tmp/repo /opt/mise/cache \
    && sed -i '1i . /etc/profile.d/orca.sh' /home/dev/.bashrc

ENTRYPOINT ["/usr/local/bin/orca-workspace-entrypoint"]
