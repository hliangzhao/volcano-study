#!/bin/bash

# Copyright 2014 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

################ Explanations ################
# Update generated code if possible.
##############################################

set -o errexit
set -o nounset
set -o pipefail

echo $(dirname "${BASH_SOURCE[0]}")

echo "For install generated codes at right position, the project root should be GOPATH/src."
echo "Or you need to manually copy the generated code to the right place."

SCRIPT_ROOT=$(unset CDPATH && cd $(dirname "${BASH_SOURCE[0]}")/.. && pwd)

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.

bash ${SCRIPT_ROOT}/hack/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/hliangzhao/volcano/pkg/client github.com/hliangzhao/volcano/pkg/apis \
  "batch:v1alpha1 bus:v1alpha1 scheduling:v1alpha1 nodeinfo:v1alpha1" \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt

echo "------------------------------"

bash ${SCRIPT_ROOT}/hack/generate-internal-groups.sh "deepcopy,conversion" \
  github.com/hliangzhao/volcano/pkg/apis/ github.com/hliangzhao/volcano/pkg/apis github.com/hliangzhao/volcano/pkg/apis \
  "scheduling:v1alpha1" \
  --go-header-file ${SCRIPT_ROOT}/hack/boilerplate.go.txt

echo "done"
