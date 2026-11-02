FROM golang:1.18-alpine AS build
# go build
ENV BUILD_DIR=/tmp/build
RUN mkdir -p ${BUILD_DIR}
COPY ./ ${BUILD_DIR}
RUN apk add --no-cache git
ENV GOPROXY=https://goproxy.cn,direct
RUN cd ${BUILD_DIR} && GOOS=linux GOARCH=amd64 go build -o notesite -ldflags "-s -w -X github.com/Streamlet/notesite/global.Version=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD 2>/dev/null || echo dev)"
# organize directories
ENV INSTALL_DIR=/usr/local/notesite
RUN mkdir -p ${INSTALL_DIR}
RUN cp ${BUILD_DIR}/notesite ${INSTALL_DIR}/
RUN cat ${BUILD_DIR}/config/site.sample.toml | \
    sed 's/#.*//' | \
    sed '/^\s*$/d' | \
    sed 's/^template_root\s*=.*$/template_root = "template"/' | \
    sed 's/note_root\s*=.*$/note_root = "note"/' \
    > ${INSTALL_DIR}/site.toml
RUN cp -R ${BUILD_DIR}/template/sample ${INSTALL_DIR}/template
RUN cp -R ${BUILD_DIR}/note/sample ${INSTALL_DIR}/note

FROM alpine:latest
COPY --from=build /usr/local/notesite /usr/local/notesite
WORKDIR /usr/local/notesite
CMD [ "./notesite" ]
