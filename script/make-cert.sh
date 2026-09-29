#!/bin/sh
# 建立一張自簽的 Code Signing 憑證並放進登入鑰匙圈,給 `make bundle` 簽 .app 用。
#
# 為什麼要有固定的憑證:macOS 的攝影機權限(TCC)是記在「程式的簽章身分」上。ad-hoc 簽章的身分
# 就是執行檔雜湊,每次重新編譯都會變,授權就作廢,所以每次啟動都要重新允許。用固定憑證簽,
# 身分不變,允許一次就一直有效。
#
# 憑證只需要建一次;已經有同名憑證就不會重複建立。
set -e

sName="${SIGN_IDENTITY:-landan-desktop-dev}"

if security find-identity -p codesigning | grep -q "\"$sName\""; then
	echo "憑證 $sName 已存在,不用再建"
	exit 0
fi

sDir="$(mktemp -d)"
trap 'rm -rf "$sDir"' EXIT

# 用設定檔而不是 -addext:macOS 內建的 LibreSSL 版本不一定支援 -addext。
cat >"$sDir/openssl.cnf" <<CNF
[req]
distinguished_name = dn
x509_extensions = ext
prompt = no
[dn]
CN = $sName
[ext]
basicConstraints = critical,CA:false
keyUsage = critical,digitalSignature
extendedKeyUsage = critical,codeSigning
CNF

openssl req -x509 -newkey rsa:2048 -nodes -days 3650 -config "$sDir/openssl.cnf" \
	-keyout "$sDir/key.pem" -out "$sDir/cert.pem"
openssl pkcs12 -export -inkey "$sDir/key.pem" -in "$sDir/cert.pem" \
	-out "$sDir/cert.p12" -passout pass:landan

# -T:允許 codesign 使用這把私鑰,簽章時才不會一直跳出鑰匙圈詢問。
security import "$sDir/cert.p12" -k "$HOME/Library/Keychains/login.keychain-db" \
	-P landan -T /usr/bin/codesign

echo "已建立憑證 $sName"
security find-identity -p codesigning | grep "$sName" || true
