#!/bin/bash
set -e

CONTAINER_NAME="openocta-server"
IMAGE_NAME="openocta:latest"
PORT=18900
DATA_DIR="$HOME/.openocta"

ACTION=$1

if [ "$ACTION" = "rebuild" ] || [ "$ACTION" = "build" ]; then
    echo "==========================================="
    echo "==> Step 1: Rebuilding Docker image..."
    echo "==========================================="
    docker build -f deploy/Dockerfile -t "$IMAGE_NAME" .
    
    echo "==========================================="
    echo "==> Step 2: Stopping and removing old container..."
    echo "==========================================="
    docker stop "$CONTAINER_NAME" 2>/dev/null || true
    docker rm "$CONTAINER_NAME" 2>/dev/null || true
    
    echo "==========================================="
    echo "==> Step 3: Starting new container..."
    echo "==========================================="
    docker run -d \
      --name "$CONTAINER_NAME" \
      -p "$PORT":18900 \
      -v "$DATA_DIR":/root/.openocta \
      "$IMAGE_NAME"
      
    echo "==> Rebuild and deploy completed!"
    echo "==> OpenOcta is running at http://127.0.0.1:$PORT"
    
elif [ "$ACTION" = "restart" ] || [ -z "$ACTION" ]; then
    echo "==========================================="
    echo "==> Restarting existing container..."
    echo "==========================================="
    if [ "$(docker ps -a -q -f name=^/${CONTAINER_NAME}$)" ]; then
        docker restart "$CONTAINER_NAME"
        echo "==> Container restarted!"
    else
        echo "==> Container '$CONTAINER_NAME' does not exist."
        echo "==> Running a new one using existing image..."
        docker run -d \
          --name "$CONTAINER_NAME" \
          -p "$PORT":18900 \
          -v "$DATA_DIR":/root/.openocta \
          "$IMAGE_NAME"
    fi
    echo "==> OpenOcta is running at http://127.0.0.1:$PORT"
else
    echo "Usage: $0 [rebuild | restart]"
    echo "  rebuild : Rebuilds the docker image and restarts container (use when frontend/backend code changes)"
    echo "  restart : Simply restarts the existing docker container"
    exit 1
fi
