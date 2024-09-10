# FHIR Server Benchmark Tool

## About

FHIR (Fast Healthcare Interoperability Resources) is a standard for healthcare data exchange. This tool benchmarks the performance of FHIR servers. It tests 3 aspects:
1. How fast can the server process uploaded FHIR resources? The benchmark will upload single FHIR resources to test processing speed.
2. How fast can it process a set of queries?
3. Will it reject invalid resources, that are not exactly well-formed?

This benchmark is designed to send resources and queries as quickly as possible to a FHIR server. The goal is for a smaller machine to fully utilize a more powerful server, thereby revealing its true capabilities.. This is implemented using Go programs that send requests concurrently. A good rule of thumb is to set the number of threads to twice the number of CPU cores available to the container running the benchmark.

A Containerfile wraps the entire benchmark into a single container that includes the necessary test data.

This is a sample output of running the benchmark container against a Blaze FHIR server with the dataset specified below on an AWS EC2 machine of the type c5.xlarge with Ubuntu Linux 24.04 on 26th June 2024. In this case the benchmark and the FHIR server are running as containers on the same machine.
```
# 1. Running upload speed test
Total uploaded resources: 134139, Total errors: 0, Total time: 2m11.793484748s
Sleeping 75 seconds to give the server time to index and map the data
# 2. Running query speed test
Total time taken: 11.471752547s, total requests sent: 9614, total errors: 0
The server responded with 161M of data to the queries.
# 3. Testing to upload invalid resources
The server rejected 10 out of 10 resources as expected.
Error code 400: 10 times
```

## How to run this benchmark
The benchmark includes data and a Containerfile to build a container for testing your FHIR server. If you follow a different procedure, please mention this in the comments of your test results.

## System Requirements
This software is designed to run on a native GNU/Linux environment. We recommend using a common Linux distribution such as Fedora, Ubuntu, or any other Linux operating system running directly on physical hardware or in a cloud-based environment. Running this software on Windows Subsystem for Linux (WSL) is not recommended due to its limitations in orchestrating containers and allocating system resources. If WSL must be used as a temporary solution, please refer to the troubleshooting section at the end of this page.

## Example workflow

In this example, we will use [Podman](https://podman.io/) to manage the containers. Podman can be easily installed on common Linux distributions and macOS using the respective package manager (apt, brew, dnf, etc.).. If you prefer, of course you can use other solutions like Docker or Kubernetes instead.

### Step 1. Clone the repository, get the test data, build the container

```bash
clone [git@ukl-git.ukl.uni-freiburg.de:translationszentrum/fhir-benchmark.git]
cd fhir-benchmark
# Download the test data
make install
# Build the container image
podman build -t fhir-benchmark-img .
```
The dataset downloaded with `make install` in the folder `./data` consists of ndjson files named after FHIR resources like `Patient.ndjson`, `Practitioner.ndjson`, `Location.ndjson` etc. and lists of the IDs of these resources in corresponding files like `Patient.ids`, `Practitioner.ids`, `Location.ids` etc. For testing purposes, there is also a folder `./invalid-data` with invalid FHIR resources.

### Step 2. Assign CPU and memory resources

To ensure consistent benchmarking, please assign the following resources:
- Benchmark container: 4 CPUs, 4 GB of RAM
- FHIR server instance: 8 CPUs, 8 GB of RAM
You can modify these settings according to your needs. If you do, please describe the settings in the tender documents. 

**Example 1. Running the container within a Podman pod**
The following snippet will create a container pod with 2 containers. You can achieve the same result with other solutions like a `docker-compose` file, a Kubernetes deployment file or a virtual machine.
```bash
# Create a Podman pod
podman pod create --name=fhir-benchmark-pod -p 8080:8080
# The next command will add a Blaze FHIR server to the pod.
# Adapt the command to add another FHIR server of your choice.
podman run \
--cpus=8 \
--memory=8G \
-d \
--pod=fhir-benchmark-pod \
--name=blaze-fhir \
docker.io/samply/blaze
# Add the benchmark container to the pod
podman run --cpus=4 --memory=4G --pod=fhir-benchmark-pod -d -it --name=fhir-benchmark fhir-benchmark-img
```

**Example 2. Running the container as a stand-alone instance**
If you prefer not to run your FHIR server within a pod as shown in Example 1, you can run the container independently.
```bash
podman run --cpus=4 --memory=4G -d -it --name=fhir-benchmark fhir-benchmark-img
```

### Step 3. Run the benchmark

You can now run the benchmark like this
```bash
podman exec fhir-benchmark benchmark.sh \
-queries_config_file=["path to queries config file"] \
-server_url=["FHIR server base URL"] \
-threads=[N] \
-upload_files=["Comma separated list of NDJSON files to upload"]
```
If a parameter is not specified, its default value will be used. Therefore, you can simply run:
```bash
podman exec fhir-benchmark benchmark.sh
```

This assumes the container can reach the server at the default address http://localhost:8080/fhir. To specify another address, run:
```bash
podman exec fhir-benchmark benchmark.sh -server_url=["FHIR server base URL"]
```

The snippet below provides a starting point for customizing the benchmark according to your needs:
```bash
# Default values:
# Server URL: http://localhost:8080/fhir
# Number of threads: 8
# NDJSON files for the FHIR resources Patient, Practitioner, Location, Organization, Encounter, Condition, Observation, Procedure, AllergyIntolerance, Immunization 
# Queries test file: queries-00.yaml
podman exec fhir-benchmark benchmark.sh \
-bearer_token="" \
-queries_config_file=queries-00.yaml \
-server_url=http://localhost:8080/fhir \
-threads=8 \
-upload_files=data/Patient.ndjson,data/Practitioner.ndjson,data/Location.ndjson,data/Organization.ndjson,data/Encounter.ndjson,data/Condition.ndjson,data/Observation.ndjson,data/Procedure.ndjson,data/AllergyIntolerance.ndjson,data/Immunization.ndjson
```

### Optional: Use Bearer Token for Authentication

If your FHIR server requires authentication via a bearer token, you can pass the token to the benchmark components using the `-bearer_token` flag. This allows the benchmark to authenticate each request made to the server.

To use a bearer token, you can pass it to the script like this:

Example
```bash
./benchmark.sh -bearer_token="your_token_here" [Everything else remains the same]
```

### Optional: Test single components

#### Upload speed test with `push-data`

For test purposes, you can also just run the single components of the benchmark.

Usage:
```bash
push-data -threads=[N] -server_url=["FHIR server base URL"] -files=["Comma separated list of files to upload"]
```
Example on host machine
```bash
podman exec fhir-benchmark push-data \
-threads=8 \
-server_url=http://localhost:8080/fhir \
-files=data/Patient.ndjson,data/Practitioner.ndjson,data/Location.ndjson,data/Organization.ndjson,data/Encounter.ndjson,data/Condition.ndjson,data/Observation.ndjson,data/Procedure.ndjson,data/AllergyIntolerance.ndjson,data/Immunization.ndjson,data/DiagnosticReport.ndjson,data/DocumentReference.ndjson,data/ImagingStudy.ndjson,data/MedicationRequest.ndjson,data/CarePlan.ndjson,data/CareTeam.ndjson,data/MedicationAdministration.ndjson,data/Claim.ndjson
```
When uploading data, be sure to list files in correct order to preserve dependencies.
1. Patient
1. Practitioner
1. Location
1. Organization
1. Medication
1. Device
1. Encounter
1. Condition
1. Observation
1. Procedure
1. AllergyIntolerance
1. Immunization
1. DiagnosticReport
1. DocumentReference
1. ImagingStudy
1. MedicationRequest
1. CarePlan
1. CareTeam
1. MedicationAdministration
1. MedicationDispense
1. Claim

#### Query speed test with `query-runner`
Usage:
```bash
query-runner -threads=[N] -config=["path to config file"] -server_url=["FHIR server base URL"]
```
Example on host machine
```bash
podman exec fhir-benchmark query-runner \
-threads=8 \
-config=queries-00.yaml \
-server_url=http://localhost:8080/fhir
```

#### Test if the server rejects invalid resources
The program `push-invalid-data` tries to upload FHIR resources, that are not well-formed, to the server. The expected result of this test is that the server rejects each of these resources with an error code.
Usage:
```bash
push-invalid-data -files=["Comma separated list of files to upload"] -server_url=["FHIR server base URL"]
```
Example:
```bash
podman exec fhir-benchmark push-invalid-data \
-server_url=http://localhost:8080/ \
-files=invalid-data/Patient.ndjson,invalid-data/Encounter.ndjson,invalid-data/Condition.ndjson,invalid-data/Observation.ndjson
```

### Monitoring the benchmark

Run `htop` to monitor CPU load and memory usage during the benchmark:
```bash
podman exec -it fhir-benchmark htop
``` 
Monitor the HAPI FHIR server logs:
```bash
podman logs -f hapi-fhir
```
To check the server, count how many resources of a certain class are on the FHIR server, e.g.
```bash
curl -X GET "http://localhost:8080/fhir/Observation?_summary=count" -H "Accept: application/fhir+json"
```

### Troubleshooting

- Step 1: If you are using WSL, please make sure that you are using version 2. 
- Step 2: If you are using WSL, add the parameter `infra=true` to the pod creation command to ensure proper initialization.