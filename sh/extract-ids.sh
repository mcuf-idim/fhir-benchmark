#!/bin/bash
set -euo pipefail

for file in $(ls data/*.ndjson); do
	echo "Extracting ids from $file to ${file%.ndjson}.ids"
	jq --raw-output '.id' <$file >${file%.ndjson}.ids
done