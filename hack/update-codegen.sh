#!/usr/bin/env bash
set -e
cd  $(dirname $0)
ROOT_PACKAGE="github.com/daicheng123/ordertask-operator"
GO111MODULE=on

[[ -d $GOPATH/src/k8s.io/code-generator ]] || go get -u k8s.io/code-generator/...

for i in deepcopy-gen client-gen;
  do
    $GOPATH/src/k8s.io/code-generator/generate-groups.sh ${i} "${ROOT_PACKAGE}/pkg/k8s" "${ROOT_PACKAGE}/pkg/apis" "tasks:v1alpha1"
  done