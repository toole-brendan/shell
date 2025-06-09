#!/bin/bash
set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$SCRIPT_DIR/../.."
RANDOMX_DIR="$PROJECT_ROOT/third_party/randomx"

echo "Building RandomX C++ library..."

# Check if RandomX submodule exists
if [ ! -d "$RANDOMX_DIR" ]; then
    echo "RandomX submodule not found. Initializing..."
    cd "$PROJECT_ROOT"
    git submodule add https://github.com/tevador/RandomX.git third_party/randomx
    cd "$RANDOMX_DIR"
    git checkout v1.2.1
fi

# Build RandomX
cd "$RANDOMX_DIR"
if [ ! -d "build" ]; then
    mkdir build
fi

cd build

# Configure based on OS
if [[ "$OSTYPE" == "darwin"* ]]; then
    cmake .. -DARCH=native -DBUILD_SHARED_LIBS=OFF -DCMAKE_C_COMPILER=clang -DCMAKE_CXX_COMPILER=clang++
else
    cmake .. -DARCH=native -DBUILD_SHARED_LIBS=OFF
fi

# Build with available cores
if [[ "$OSTYPE" == "darwin"* ]]; then
    make -j$(sysctl -n hw.ncpu)
else
    make -j$(nproc)
fi

echo "RandomX build complete!" 