#!/bin/bash

set -e  # exit immediately on error
set -x  # display all commands

if [ ! -f kubebuilder ]; then
	if [ ! -f kubebuilder_2.2.0_darwin_amd64.tar.gz ]; then
		wget https://github.com/kubernetes-sigs/kubebuilder/releases/download/v2.2.0/kubebuilder_2.2.0_darwin_amd64.tar.gz
	fi

	tar -xf kubebuilder_2.2.0_darwin_amd64.tar.gz
    cp kubebuilder_2.2.0_darwin_amd64/bin/kubebuilder ./
    chmod a+x ./kubebuilder
fi



echo "all done."