#!/bin/sh
set -e -x

if [ "$#" -lt 1 ]; then
    echo "Usage: $0 ARGS"
    exit 1
fi

if [ -z "${SBP_IMAGE_NAME}" ]; then
    echo "SBP_IMAGE_NAME needs to be set (string)"
    exit 1
fi

if [ -z "${SBP_PUSH}" ]; then
    echo "SBP_PUSH needs to be set (0 or 1)"
    exit 1
fi

s2i build $@
if [ "${SBP_PUSH}" = 1 ]; then
    docker push ${SBP_IMAGE_NAME}
fi
