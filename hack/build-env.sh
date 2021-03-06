#!/bin/bash

# Copyright 2020 The Volcano Authors.

# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at

#     http://www.apache.org/licenses/LICENSE-2.0

# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

################ Explanations ################
# This script is used to build dev and release environments.
# Specifically, it sets different values of GitSHA and RELEASE_VER.
##############################################

ENV=$1

GitSHA=$(git rev-parse HEAD)

cp defs/Makefile."${ENV}".def Makefile.def

# TODO: NOTE that "" is only required for macOS! Remove it when uploading the code to linux.
sed -i "" "s/__git_sha__/${GitSHA}/g" Makefile.def
sed -i "" "s/__release_ver__/${RELEASE_VER}/g" Makefile.def
