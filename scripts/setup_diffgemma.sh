#!/usr/bin/env bash
set -euo pipefail

echo "==> Checking prerequisites for DiffusionGemma (diffgemma)..."

# Verify OS
if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "Error: diffgemma requires macOS on Apple Silicon."
  exit 1
fi

# Verify architecture
if [[ "$(uname -m)" != "arm64" ]]; then
  echo "Error: Apple Silicon (arm64) is required for Metal acceleration."
  exit 1
fi

# Verify Rust / Cargo
if ! command -v cargo >/dev/null 2>&1; then
  echo "Error: cargo not found. Please install Rust via https://rustup.rs"
  exit 1
fi

echo "==> Building and installing diffgemma from upstream git repository..."
cargo install --git https://github.com/mmastrac/diffgemma diffgemma

echo "==> diffgemma installed successfully: $(command -v diffgemma)"
