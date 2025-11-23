#!/bin/bash

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

echo "Building tidy..."
go build -o tidy main.go

if [ $? -ne 0 ]; then
    echo "Build failed."
    exit 1
fi

INSTALL_DIR="/usr/local/bin"

echo "Installing to $INSTALL_DIR..."
# Check if we have write permissions to the install directory
if [ -w "$INSTALL_DIR" ]; then
    mv tidy "$INSTALL_DIR/"
else
    echo "Sudo permissions required to move binary to $INSTALL_DIR"
    sudo mv tidy "$INSTALL_DIR/"
fi

echo "Successfully installed tidy to $INSTALL_DIR/tidy"
