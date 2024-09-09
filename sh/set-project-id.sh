#!/bin/bash
set -euo pipefail

# Default values
THREADS=32
INPUT_DIR=""
OUTPUT_DIR=""
PROJECT_ID=""

# Function to print usage
usage() {
    echo "Usage: $0 -input_dir=[input directory] -output_dir=[output directory] -project_id=[project id to be added] [-threads=[max number of source files processed concurrently]]"
    exit 1
}

# Function to process a single ndjson file
process_file() {
    local input_file="$1"
    local output_file="$2"
    local project_id="$3"

    [[ ! -f "$output_file" ]] || rm "$output_file"

    # Process each line of the ndjson file
    while IFS= read -r line || [[ -n "$line" ]]; do
        # Skip empty lines
        if [[ -z "$line" ]]; then
            continue
        fi

        # Use jq to add or update the meta.project field and output as compact JSON (one line)
        modified_line=$(echo "$line" | jq --arg proj_id "$project_id" \
                        '.meta.project = $proj_id | .meta |= . + {"project": $proj_id}' -c)

        # Append the modified line to the output file
        echo "$modified_line" >> "$output_file"
    done < "$input_file"
}


export -f process_file

# Parse arguments using if-else statements
for arg in "$@"; do
    if [[ "$arg" == -input_dir=* ]]; then
        INPUT_DIR="${arg#*=}"
    elif [[ "$arg" == -output_dir=* ]]; then
        OUTPUT_DIR="${arg#*=}"
    elif [[ "$arg" == -project_id=* ]]; then
        PROJECT_ID="${arg#*=}"
    elif [[ "$arg" == -threads=* ]]; then
        THREADS="${arg#*=}"
    else
        usage
    fi
done

# Check if required arguments are provided
if [[ -z "$INPUT_DIR" || -z "$OUTPUT_DIR" || -z "$PROJECT_ID" ]]; then
    usage
fi

# Create the output directory if it doesn't exist
mkdir -p "$OUTPUT_DIR"

# Find all ndjson files in the input directory and process them in parallel
find "$INPUT_DIR" -type f -name "*.ndjson" | parallel -j "$THREADS" process_file {} "$OUTPUT_DIR/{/}" "$PROJECT_ID"
