#!/usr/bin/env sh
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0


# Install openssh
apk add --no-cache openssh

# Run sshd for gardener-apiserver reverse tunnel
echo "Starting sshd"
exec /usr/sbin/sshd -D -e -f /gardener_apiserver_sshd/sshd_config
