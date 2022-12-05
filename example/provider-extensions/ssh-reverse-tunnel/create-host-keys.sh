#!/usr/bin/env bash
#
# Copyright (c) 2022 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses~LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

usage() {
  echo "Usage:"
  echo "> create-host-keys.sh [ -h | <host> <port> ]"
  echo
  echo ">> For example: create-host-keys.sh localhost 22"

  exit 0
}

if [ "$1" == "-h" ] || [ "$#" -ne 2 ]; then
  usage
fi

host=$1
port=$2

ssh-keygen -q -C "" -N "" -t rsa -b 4096 -f "$SCRIPT_DIR"/sshd/host-keys/ssh_host_rsa_key <<< y >/dev/null
ssh-keygen -q -C "" -N "" -t ecdsa -f "$SCRIPT_DIR"/sshd/host-keys/ssh_host_ecdsa_key <<< y >/dev/null
ssh-keygen -q -C "" -N "" -t ed25519 -f "$SCRIPT_DIR"/sshd/host-keys/ssh_host_ed25519_key <<< y >/dev/null

rm -rf "$SCRIPT_DIR"/ssh/client-keys/known_hosts

{
    echo "[$host]:$port $(cat "$SCRIPT_DIR"/sshd/host-keys/ssh_host_rsa_key.pub)"
    echo "[$host]:$port $(cat "$SCRIPT_DIR"/sshd/host-keys/ssh_host_ecdsa_key.pub)"
    echo "[$host]:$port $(cat "$SCRIPT_DIR"/sshd/host-keys/ssh_host_ed25519_key.pub)"
} >> "$SCRIPT_DIR"/ssh/client-keys/known_hosts

echo "$host" > "$SCRIPT_DIR"/ssh/client-keys/host
