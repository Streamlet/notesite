FROM golang:1.17-alpine AS build
# go build
ENV BUILD_DIR=/tmp/build
RUN mkdir -p ${BUILD_DIR}
COPY ./ ${BUILD_DIR}
ENV GOPROXY=https://goproxy.cn,direct
RUN cd ${BUILD_DIR} && GOOS=linux GOARCH=amd64 go build -o notesite -ldflags='-s -w'
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
