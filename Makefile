archive_urls := https://raw.githubusercontent.com/mcuf-idim/fhir-benchmark-data/main/data.tar.gz \
                https://raw.githubusercontent.com/mcuf-idim/fhir-benchmark-data/main/invalid-data.tar.gz
go_files := $(wildcard src/*.go)
go_programs := $(patsubst src/%.go,%,${go_files})

all: ${go_programs}

%: src/%.go
	go build -o $@ $<

container-img: fhir-benchmark-img.tar.gz

fhir-benchmark-img.tar.gz: Containerfile Makefile README.md data src .dockerignore
	podman build -t fhir-benchmark-img .; \
	podman save -o fhir-benchmark-img.tar fhir-benchmark-img; \
	gzip fhir-benchmark-img.tar;

install:
	@for url in ${archive_urls}; do \
		filename=$$(basename $${url} .tar.gz); \
		echo "Downloading $${url}..."; \
		curl -L $${url} -o $${filename}.tar.gz; \
		echo "Creating directory $${filename}..."; \
		mkdir -p $${filename}; \
		echo "Unpacking $${filename}.tar.gz..."; \
		tar -xzf $${filename}.tar.gz -C $${filename}; \
		echo "Cleaning up..."; \
		rm $${filename}.tar.gz; \
	done
clean:
	rm -f push-data push-invalid-data query-runner fhir-benchmark-img.tar.gz

.PHONY: all clean hapi-fhir-logs hapi-fhir-pull hapi-fhir-reset hapi-fhir-run install