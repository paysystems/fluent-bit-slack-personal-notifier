#!/bin/bash

set -euo pipefail

PLUGINS_DIR="plugins"
DIST_DIR="dist"

mkdir -p "$DIST_DIR"
cd "$PLUGINS_DIR"

echo "Starting compilation of Fluent Bit plugins."

for dir in */; do
    if [[ -d "$dir" ]]; then
        dir_name=$(basename "$dir")
        echo "-> Processing plugin: $dir_name"

        cd "$dir"
        go mod download
        go build -buildmode=c-shared -o "${dir_name}.so"
        mv ./*.so ./*.h "../../$DIST_DIR/"
        cd ..
    fi
done

echo "All plugins processed. Compiled files are saved in the $DIST_DIR directory."