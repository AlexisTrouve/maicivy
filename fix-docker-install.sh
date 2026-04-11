#!/bin/bash

echo "================================"
echo "Fixing Docker Installation"
echo "================================"

# Try to fix the broken installation
echo "⏳ Fixing broken packages..."
sudo apt --fix-broken install -y

# Try installing with --fix-missing
echo "⏳ Retrying Docker installation with --fix-missing..."
sudo apt install -y --fix-missing docker.io docker-compose

# If still failing, try alternative approach
if ! command -v docker &> /dev/null; then
    echo "⏳ Trying alternative installation method..."

    # Install from Docker's official repository
    echo "⏳ Installing prerequisites..."
    sudo apt install -y ca-certificates curl gnupg lsb-release

    echo "⏳ Adding Docker's official GPG key..."
    sudo mkdir -p /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg

    echo "⏳ Setting up Docker repository..."
    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    echo "⏳ Updating package list..."
    sudo apt update

    echo "⏳ Installing Docker from official repository..."
    sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

# Add user to docker group
echo "⏳ Adding user to docker group..."
sudo usermod -aG docker $USER

# Start Docker service
echo "⏳ Starting Docker service..."
sudo service docker start

# Verify installation
echo ""
echo "================================"
echo "Verifying Docker installation..."
echo "================================"
docker --version
docker compose version

# Test Docker
echo ""
echo "================================"
echo "Testing Docker..."
echo "================================"
docker run --rm hello-world

echo ""
echo "================================"
echo "✅ Docker is ready!"
echo "================================"
echo ""
echo "Run: newgrp docker"
echo "Then: ./start-project.sh"
