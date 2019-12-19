#!/bin/bash

set -e  # exit immediately on error
set -x  # display all commands

tools/kubebuilder init --domain dmall.com --license apache2 --owner "The dks authors"
tools/kubebuilder create api --group workload --version v1beta1 --kind AdvDeployment

echo "all done."