#!/usr/bin/env bash
# Trust this runner's CA only in the generated emulator build or the booted simulator.
set -euo pipefail
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
umask 077
tls="$RUNNER_TEMP/mobile-tls"
mkdir "$tls"
cat > "$tls/ca.cnf" <<'CONFIG'
[req]
distinguished_name=dn
x509_extensions=ca
prompt=no
[dn]
CN=EduPilot disposable mobile fixture
[ca]
basicConstraints=critical,CA:TRUE
keyUsage=critical,keyCertSign,cRLSign
CONFIG
openssl req -new -x509 -nodes -newkey rsa:2048 -days 2 -sha256 -config "$tls/ca.cnf" -keyout "$tls/ca.key" -out "$tls/ca.pem"
openssl req -new -nodes -newkey rsa:2048 -subj /CN=localhost -keyout "$tls/server.key" -out "$tls/server.csr"
printf '%s\n' 'basicConstraints=critical,CA:FALSE' 'keyUsage=digitalSignature,keyEncipherment' 'extendedKeyUsage=serverAuth' 'subjectAltName=DNS:localhost,IP:127.0.0.1' > "$tls/ext.cnf"
openssl x509 -req -in "$tls/server.csr" -CA "$tls/ca.pem" -CAkey "$tls/ca.key" -CAcreateserial -days 1 -sha256 -extfile "$tls/ext.cnf" -out "$tls/server.pem"
if [[ "$1" = android ]]; then
  src=clients/mobile/android/app/src/main
  mkdir -p "$src/res/raw" "$src/res/xml"
  cp "$tls/ca.pem" "$src/res/raw/fixture_ca.pem"
  cat > "$src/res/xml/mobile_fixture_trust.xml" <<'XML'
<network-security-config>
  <domain-config cleartextTrafficPermitted="false">
    <domain includeSubdomains="false">localhost</domain>
    <trust-anchors><certificates src="@raw/fixture_ca"/></trust-anchors>
  </domain-config>
</network-security-config>
XML
  sed -i 's/<application /<application android:networkSecurityConfig="@xml\/mobile_fixture_trust" /' "$src/AndroidManifest.xml"
  grep -q 'networkSecurityConfig="@xml/mobile_fixture_trust"' "$src/AndroidManifest.xml"
else
  xcrun simctl keychain booted add-root-cert "$tls/ca.pem"
fi
