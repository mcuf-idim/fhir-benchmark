#!/bin/bash
set -euo pipefail

total_lines=0
total_valid_json=0

if [ "$#" -eq 0 ]; then
  echo "Usage: $0 -files=[list of files]"
  exit 1
fi

files=""
for arg in "$@"; do
  case $arg in
    -files=*)
      files="${arg#-files=}"
      shift
      ;;
    *)
      echo "Unknown option: $arg"
      echo "Usage: $0 -files=[list of files]"
      exit 1
      ;;
  esac
done

if [ -z "$files" ]; then
  echo "No files specified."
  echo "Usage: $0 -files=[list of files]"
  exit 1
fi

echo "Counts per file:"

IFS=',' read -r -a files_array <<<"$files"

for file in "${files_array[@]}"; do
  file=$(echo "$file" | xargs)
  if [ -f "$file" ]; then
    total_lines_in_file=$(wc -l < "$file")
    valid_json_lines_in_file=$(jq -c . <"$file" 2>/dev/null | wc -l)
    echo "$file: total lines = $total_lines_in_file, valid JSON objects = $valid_json_lines_in_file"
    total_lines=$((total_lines + total_lines_in_file))
    total_valid_json=$((total_valid_json + valid_json_lines_in_file))
  else
    echo "File not found: $file"
  fi
done

echo
echo "Total lines across all files: $total_lines"
echo "Total valid JSON objects: $total_valid_json"
