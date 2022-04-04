#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

{
SECRET_NAME="{{ .secretName }}"
TOKEN_SECRET_NAME="{{ .tokenSecretName }}"

PATH_CLOUDCONFIG_DOWNLOADER_CLIENT_CERT="{{ .pathCredentialsClientCert }}"
PATH_CLOUDCONFIG_DOWNLOADER_CLIENT_KEY="{{ .pathCredentialsClientKey }}"
PATH_BOOTSTRAP_TOKEN="{{ .pathBootstrapToken }}"
PATH_CLOUDCONFIG_DOWNLOADER_TOKEN="{{ .pathCredentialsToken }}"

function readSecret() {
  wget \
    -qO- \
    --header         "Accept: application/yaml" \
    --ca-certificate "{{ .pathCredentialsCACert }}" \
    "${@:2}" "$(cat "{{ .pathCredentialsServer }}")/api/v1/namespaces/kube-system/secrets/$1"
}

function readSecretWithToken() {
  readSecret "$1" "--header=Authorization: Bearer $2"
}

function readSecretWithClientCertificate() {
  readSecret "$1" "--certificate=$PATH_CLOUDCONFIG_DOWNLOADER_CLIENT_CERT" "--private-key=$PATH_CLOUDCONFIG_DOWNLOADER_CLIENT_KEY"
}

function extractDataKeyFromSecret() {
  echo "$1" | sed -rn "s/  $2: (.*)/\1/p" | base64 -d
}

function writeToDiskSafely() {
  local data="$1"
  local file_path="$2"

  if echo "$data" > "$file_path.tmp" && ( [[ ! -f "$file_path" ]] || ! diff "$file_path" "$file_path.tmp">/dev/null ); then
    mv "$file_path.tmp" "$file_path"
  elif [[ -f "$file_path.tmp" ]]; then
    rm -f "$file_path.tmp"
  fi
}

# download shoot access token for cloud-config-downloader
if [[ -f "$PATH_CLOUDCONFIG_DOWNLOADER_TOKEN" ]]; then
  if ! SECRET="$(readSecretWithToken "$TOKEN_SECRET_NAME" "$(cat "$PATH_CLOUDCONFIG_DOWNLOADER_TOKEN")")"; then
    echo "Could not retrieve the shoot access secret with name $TOKEN_SECRET_NAME with existing token"
    exit 1
  fi
else
  if [[ -f "$PATH_BOOTSTRAP_TOKEN" ]]; then
    if ! SECRET="$(readSecretWithToken "$TOKEN_SECRET_NAME" "$(cat "$PATH_BOOTSTRAP_TOKEN")")"; then
      echo "Could not retrieve the shoot access secret with name $TOKEN_SECRET_NAME with bootstrap token"
      exit 1
    fi
  else
    if ! SECRET="$(readSecretWithClientCertificate "$TOKEN_SECRET_NAME")"; then
      echo "Could not retrieve the shoot access secret with name $TOKEN_SECRET_NAME with client certificate"
      exit 1
    fi
  fi
fi

TOKEN="$(extractDataKeyFromSecret "$SECRET" "{{ .dataKeyToken }}")"
if [[ -z "$TOKEN" ]]; then
  echo "Token in shoot access secret $TOKEN_SECRET_NAME is empty"
  exit 1
fi

writeToDiskSafely "$TOKEN" "$PATH_CLOUDCONFIG_DOWNLOADER_TOKEN"

# download and run the cloud config execution script
if ! SECRET="$(readSecretWithToken "$SECRET_NAME" "$TOKEN")"; then
  echo "Could not retrieve the cloud config script in secret with name $SECRET_NAME"
  exit 1
fi

CHECKSUM="$(echo "$SECRET" | sed -rn 's/    {{ .annotationChecksum | replace "/" "\\/" }}: (.*)/\1/p' | sed -e 's/^"//' -e 's/"$//')"
echo "$CHECKSUM" > "{{ .pathDownloadedChecksum }}"

SCRIPT="$(extractDataKeyFromSecret "$SECRET" "{{ .dataKeyScript }}")"
echo "$SCRIPT" | bash

exit $?
}
