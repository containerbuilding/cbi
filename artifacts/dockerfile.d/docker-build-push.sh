#!/bin/sh
set -e -x

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 BUILDFLAGS"
    exit 1
fi

if [ ! "${DBP_DOCKER_BINARY}" ]; then
    echo "DBP_DOCKER_BINARY needs to be set (string)"
    exit 1
fi
docker=${DBP_DOCKER_BINARY}


if [ ! "${DBP_IMAGE_NAME}" ]; then
    echo "DBP_IMAGE_NAME needs to be set (string)"
    exit 1
fi

if [ ! "${DBP_PUSH}" ]; then
    echo "DBP_PUSH needs to be set (0 or 1)"
    exit 1
fi

${docker} build -t ${DBP_IMAGE_NAME} $@

if [ "${DBP_PUSH}" = 1 ]; then
    ${docker} push ${DBP_IMAGE_NAME}
fi
