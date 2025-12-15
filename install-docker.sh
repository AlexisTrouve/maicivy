#!/bin/bash

echo "================================"
echo "Installing Docker in WSL2"
echo "================================"

# Update package list
echo "⏳ Updating package list..."
sudo apt update

# Install Docker and Docker Compose
echo "⏳ Installing Docker and Docker Compose..."
sudo apt install -y docker.io docker-compose

# Add current user to docker group
echo "⏳ Adding user to docker group..."
sudo usermod -aG docker $USER

# Start Docker service
echo "⏳ Starting Docker service..."
sudo service docker start

# Test Docker installation
echo ""
echo "✅ Docker installation complete!"
echo ""
echo "Versions installed:"
docker --version
docker-compose --version

echo ""
echo "================================"
echo "Testing Docker..."
echo "================================"
docker run hello-world

echo ""
echo "================================"
echo "✅ Docker is ready!"
echo "================================"
echo ""
echo "IMPORTANT: You may need to logout/login for group changes to take effect"
echo "Run: newgrp docker"
echo ""
echo "To start Docker service in the future:"
echo "  sudo service docker start"
echo ""
echo "Next steps:"
echo "  cd /mnt/c/Users/alexi/Documents/projects/maicivy"
echo "  docker-compose up -d postgres redis"
