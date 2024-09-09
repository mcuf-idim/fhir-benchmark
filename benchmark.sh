#!/bin/bash
set -euo pipefail

bearer_token=""
queries_config_file=queries-00.yaml
server_url="http://localhost:8080/fhir"
threads=8
upload_files="data/Patient.ndjson,data/Practitioner.ndjson,data/Location.ndjson,data/Organization.ndjson,data/Encounter.ndjson,data/Condition.ndjson,data/Observation.ndjson,data/Procedure.ndjson,data/AllergyIntolerance.ndjson,data/Immunization.ndjson"

for arg in "$@"; do
	if [[ $arg == "-bearer_token="* ]]; then
		bearer_token="${arg#-bearer_token=}"
	elif [[ $arg == "-queries_config_file="* ]]; then
		queries_config_file=${arg#-queries_config_file=}
	elif [[ $arg == "-server_url="* ]]; then
		server_url=${arg#-server_url=}
	elif [[ $arg == "-threads="* ]]; then
		threads=${arg#-threads=}
	elif [[ $arg == "-upload_files="* ]]; then
		upload_files=${arg#-upload_files=}
	else
		echo "Error. Unknown argument $arg."
		exit 1
	fi
done

echo "# 1. Running upload speed test"
push-data -bearer_token="$bearer_token" -threads=$threads -server_url=$server_url -files=$upload_files
echo "Sleeping 75 seconds to give the server time to index and map the data"
sleep 75
echo "# 2. Running query speed test"
rm -f query-output/*
query-runner -config=$queries_config_file -server_url=$server_url -threads=$threads
echo "The server responded with $(du -sh query-output | awk '{print $1}') of data to the queries."
echo "# 3. Testing to upload invalid resources"
push-invalid-data -server_url=$server_url -files=invalid-data/Patient.ndjson,invalid-data/Encounter.ndjson,invalid-data/Condition.ndjson,invalid-data/Observation.ndjson
