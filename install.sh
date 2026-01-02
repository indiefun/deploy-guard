#!/bin/bash
set -e

REPO="indiefun/deploy-guard"

# 1. 检测系统与架构
OS="$(uname -s)"
if [ "$OS" != "Linux" ]; then
    echo "Error: This script only supports Linux."
    exit 1
fi

ARCH="$(uname -m)"
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

echo "Detected system: Linux ($ARCH)"

# 2. 获取最新版本 Tag
echo "Fetching latest version..."
LATEST_URL="https://github.com/$REPO/releases/latest"
# 利用 curl 获取重定向后的 URL，从中提取 tag (例如 v1.0.0)
TAG=$(curl -sL -o /dev/null -w %{url_effective} "$LATEST_URL" | rev | cut -d/ -f1 | rev)

if [ -z "$TAG" ]; then
    echo "Error: Failed to fetch the latest version tag from $LATEST_URL"
    exit 1
fi

echo "Latest version: $TAG"

# 3. 构造下载 URL 并下载
# 命名规则参考 README: dg_<tag>_linux_<arch>.tar.gz
FILENAME="dg_${TAG}_linux_${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$TAG/$FILENAME"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

echo "Downloading $DOWNLOAD_URL ..."
curl -sL "$DOWNLOAD_URL" -o "$TMP_DIR/$FILENAME"

# 4. 解压并安装
echo "Extracting..."
tar -xzf "$TMP_DIR/$FILENAME" -C "$TMP_DIR"

# 查找二进制文件 (假设解压后文件名为 dg)
BINARY_PATH=$(find "$TMP_DIR" -type f -name dg | head -n 1)

if [ -z "$BINARY_PATH" ]; then
    echo "Error: Binary 'dg' not found in the archive."
    exit 1
fi

echo "Installing to /usr/local/bin/dg (requires sudo)..."
sudo install -m 0755 "$BINARY_PATH" /usr/local/bin/dg

echo "Success! Installed version:"
dg version
