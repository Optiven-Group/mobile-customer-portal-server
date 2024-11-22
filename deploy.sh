#!/bin/bash

# Navigate to the project directory
cd /root/mobile-customer-portal-server

# Pull the latest code
git pull origin main

# Build the application
GOOS=linux GOARCH=amd64 go build -o mobile-customer-portal-server

# Restart the service
systemctl restart customer-portal-server

echo "Deployment successful!"
