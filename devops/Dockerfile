ARG BASE_IMAGE=sberworks.ru/sbt/ci90000318_sbtws/base_images/alpine:3.21-sbt
ARG APPNAME=app

FROM ${BASE_IMAGE}
COPY . /build
WORKDIR /build

ARG OSC_TOKEN=123
ARG APP_VERSION=1.2.3
ARG BUILD_NUMBER=1
ARG APP_BRANCH=develop_sc
ARG NPM_CONFIG=npm_config.json
ARG GO_VERSION=1.23.6-r0
ARG NODE_VERSION=18.20.1-r0
ARG NPM_VERSION=10.9.1-r0
ARG SKIP_TESTS=false

RUN apk update && apk add --no-cache go=${GO_VERSION} make git && apk update --repository https://sberworks.ru/osc/repo/apk/v3.18/main --repository https://sberworks.ru/osc/repo/apk/v3.18/community && apk add --no-cache --repository https://sberworks.ru/osc/repo/apk/v3.18/main --repository https://sberworks.ru/osc/repo/apk/v3.18/community nodejs=${NODE_VERSION} npm=${NPM_VERSION}

# ENV VARS FOR GO BUILD
ENV GOPROXY="https://token:${OSC_TOKEN}@sberworks.ru/osc/repo/go"
ENV EXECUTABLE=sc
ENV EXTRA_GOFLAGS="-trimpath"
ENV GITEA_VERSION="${APP_VERSION}-${BUILD_NUMBER} (${APP_BRANCH})"
ENV GOSUMDB=off
ENV GO111MODULE=on
ENV TAGS="netgo osusergo bindata"
ENV TEST_TAGS="netgo osusergo bindata"

COPY ${NPM_CONFIG} /root/.npmrc

# BUILD
RUN make clean-all
RUN if [ "$SKIP_TESTS" = "false" ]; then CGO_ENABLED=0 make test-backend-correct; fi
RUN make frontend
RUN CGO_ENABLED=0 make backend
RUN CGO_ENABLED=0 go build contrib/environment-to-ini/environment-to-ini.go
