# Базовый образ будет определен динамически на основе /etc/os-release
ARG BASE_IMAGE=sberworks.ru/sbt/ci90000655_biprdct/slo/sberlinux-8-x86_64-ubi:8.10.0-327_20240712.0-39
FROM ${BASE_IMAGE} as base

# Проверяем систему и устанавливаем соответствующие пакеты
RUN if [ -f /etc/os-release ]; then \
        . /etc/os-release; \
        if [ "$ID" = "altlinux" ]; then \
            echo "Detected ALT Linux, installing packages via apt"; \
            apt-get update && apt-get install -y \
              ca-certificates \
              ca-certificates-digital.gov.ru \
              gettext \
              git \
              curl \
              gnupg && \
            apt-get clean && rm -rf /var/lib/apt/lists/*; \
        elif [ "$ID" = "alpine" ]; then \
            echo "Detected Alpine Linux, installing packages via apk"; \
            apk add --no-cache \
                ca-certificates \
                gettext \
                git \
                curl \
                gnupg \
                bash \
                shadow; \
        else \
            echo "Detected SberLinux/RHEL-based, installing packages via dnf"; \
            dnf upgrade -y && dnf install -y \
                ca-certificates \
                gettext \
                git \
                curl \
                gnupg && \
            dnf clean all && dnf autoremove; \
        fi; \
    else \
        echo "Warning: /etc/os-release not found, assuming SberLinux/RHEL-based"; \
        dnf upgrade -y && dnf install -y \
            ca-certificates \
            gettext \
            git \
            curl \
            gnupg && \
        dnf clean all && dnf autoremove; \
    fi

# Общая конфигурация (работает для обеих систем)
EXPOSE 22 3000

# Добавляем пользователя и группу
RUN groupadd \
    -r -g 10001 \
    git && \
    useradd \
    -r -M \
    -d /var/lib/gitea/git \
    -s /bin/bash \
    -u 10001 \
    -g git \
    git

# Создаем структуру каталогов
RUN mkdir -p /var/lib/gitea /etc/gitea
RUN chown git:git /var/lib/gitea /etc/gitea

# Копируем артефакты
COPY docker/gitt/rootless /
COPY docker/gitt/bash_autocomplete /etc/profile.d/gitea_bash_autocomplete.sh
COPY --chown=git:git bh/sc /app/gitea/gitea
COPY --chown=git:git bh/sc-gitaly-backup /app/gitea/gitea-gitaly-backup
COPY --chown=git:git docker/gitt/environment-to-ini /usr/local/bin/environment-to-ini
RUN chmod 755 /usr/local/bin/docker-entrypoint.sh /usr/local/bin/docker-setup.sh /app/gitea/gitea /usr/local/bin/gitea /usr/local/bin/environment-to-ini
RUN chmod 644 /etc/profile.d/gitea_bash_autocomplete.sh

# Устанавливаем переменные окружения
USER 10001:10001
ENV GITEA_WORK_DIR /var/lib/gitea
ENV GITEA_CUSTOM /var/lib/gitea/custom
ENV GITEA_TEMP /tmp/gitea
ENV TMPDIR /tmp/gitea
ENV GITEA_APP_INI /etc/gitea/app.ini
ENV HOME "/var/lib/gitea/git"
VOLUME ["/var/lib/gitea", "/etc/gitea"]
WORKDIR /var/lib/gitea

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD []
