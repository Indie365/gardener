#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2019 SAP SE or an SAP affiliate company and Gardener contributors
# SPDX-License-Identifier: Apache-2.0

set -e

echo "> Clean"

for source_tree in $@; do
  find "$(dirname "$source_tree")" -type f -name "zz_*.go" -exec rm '{}' \;
  find "$(dirname "$source_tree")" -type f -name "generated.proto" -exec rm '{}' \;
  find "$(dirname "$source_tree")" -type f -name "generated.pb.go" -exec rm '{}' \;
  find "$(dirname "$source_tree")" -type f -name "openapi_generated.go" -exec rm '{}' \;
  grep -lr '// Code generated by MockGen. DO NOT EDIT' "$(dirname "$source_tree")" | xargs rm -f
  grep -lr '// Code generated by client-gen. DO NOT EDIT' "$(dirname "$source_tree")" | xargs rm -f
  grep -lr '// Code generated by informer-gen. DO NOT EDIT' "$(dirname "$source_tree")" | xargs rm -f
  grep -lr '// Code generated by lister-gen. DO NOT EDIT' "$(dirname "$source_tree")" | xargs rm -f
done

if [ -d "$PWD/docs/api-reference" ]; then
  find ./docs/api-reference/ -type f -name "*.md" ! -name "README.md" -exec rm '{}' \;
fi
