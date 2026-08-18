#!/bin/bash
# 请在仓库根目录运行；第二个参数为目标平台（arm64 / amd64）。
set -euo pipefail
SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "$SCRIPT_DIR"
IMAGE_NAME=${1:-my-go-task}
PLATFORM=${2:-linux/amd64}

docker buildx build --platform "$PLATFORM" -f benzhi.Dockerfile -t "$IMAGE_NAME" .

echo ""
echo "✅ Docker image '$IMAGE_NAME' built successfully!"
echo "📋 进入容器: docker run -it $IMAGE_NAME bash"
