IMAGE_NAME=$(shell basename $(CURDIR)):latest
CONTAINER_NAME=$(shell basename $(CURDIR))_app

.PHONY: build http message command model domain migration-postgres inbound-http inbound-message-rabbitmq inbound-command outbound-database-postgres outbound-http outbound-message-rabbitmq outbound-cache-redis run generate-mocks lint test test-coverage test-integration

build:
	@if [ "$(BUILD)" = "true" ]; then \
		echo "[INFO] BUILD=true, force rebuilding Docker image $(IMAGE_NAME)..."; \
		docker build -t $(IMAGE_NAME) .; \
	elif ! docker image inspect $(IMAGE_NAME) > /dev/null 2>&1; then \
		echo "[INFO] Docker image $(IMAGE_NAME) not found. Building..."; \
		docker build -t $(IMAGE_NAME) .; \
	else \
		echo "[INFO] Docker image $(IMAGE_NAME) already exists. Skipping build."; \
	fi

http:
	$(MAKE) build BUILD=$(BUILD)
	@echo "[INFO] Running the application in HTTP server mode inside Docker."
	docker run --rm \
	  --name $(CONTAINER_NAME) \
	  --env-file .env \
	  -p 8000:8000 \
	  --network $(shell basename $(CURDIR))_default \
	  $(IMAGE_NAME) http

message:
	$(MAKE) build BUILD=$(BUILD)
	@if [ -z "$(SUB)" ]; then \
	  echo "[ERROR] Please provide SUB, e.g. make message SUB=upsert_client"; \
	  exit 1; \
	fi
	@echo "[INFO] Running the application in message mode inside Docker with argument: $(SUB)"
	docker run --rm \
	  --name $(CONTAINER_NAME)_message \
	  --env-file .env \
	  --network $(shell basename $(CURDIR))_default \
	  $(IMAGE_NAME) message $(SUB)

command:
	$(MAKE) build BUILD=$(BUILD)
	@if [ -z "$(CMD)" ] || [ -z "$(VAL)" ]; then \
	  echo "[ERROR] Please provide CMD and VAL, e.g. make command CMD=publish_upsert_client VAL=name"; \
	  exit 1; \
	fi
	@echo "[INFO] Running the application in command mode inside Docker with arguments: $(CMD) $(VAL)"
	docker run --rm \
	  --name $(CONTAINER_NAME)_command \
	  --env-file .env \
	  --network $(shell basename $(CURDIR))_default \
	  $(IMAGE_NAME) $(CMD) $(VAL)

model:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make model VAL=name"; \
		exit 1; \
	fi; \
	LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		UPPER=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		UPPER=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/model/$(VAL).go; \
	if [ -f "$$DST" ]; then \
		echo "[ERROR] File $$DST already exists."; \
		exit 1; \
	fi; \
	printf "package model\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "import (\n" >> $$DST; \
	printf "\t\"time\"\n" >> $$DST; \
	printf ")\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "type $$UPPER struct {\n" >> $$DST; \
	printf "\tID int \`json:\"id\" db:\"id\"\`\n" >> $$DST; \
	printf "\t$${UPPER}Input\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "type $${UPPER}Input struct {\n" >> $$DST; \
	printf "\tCreatedAt time.Time \`json:\"created_at\" db:\"created_at\"\`\n" >> $$DST; \
	printf "\tUpdatedAt time.Time \`json:\"updated_at\" db:\"updated_at\"\`\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "type $${UPPER}Filter struct {\n" >> $$DST; \
	printf "\tIDs []int \`json:\"ids\"\`\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "func $${UPPER}Prepare(v *$${UPPER}Input) {\n" >> $$DST; \
	printf "\tv.CreatedAt = time.Now()\n" >> $$DST; \
	printf "\tv.UpdatedAt = time.Now()\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "func (c $${UPPER}Filter) IsEmpty() bool {\n" >> $$DST; \
	printf "\tif len(c.IDs) == 0 {\n" >> $$DST; \
	printf "\t\treturn true\n" >> $$DST; \
	printf "\t}\n" >> $$DST; \
	printf "\treturn false\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	echo "[INFO] Created model file: $$DST"

domain:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make domain VAL=product"; \
		exit 1; \
	fi; \
	LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_"} {printf "%s", $$1; for(i=2;i<=NF;i++) printf "%s", toupper(substr($$i,1,1)) substr($$i,2)} END{print ""}'); \
	else \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
		CAMEL=$$LOWER; \
	fi; \
	DOMAIN_DIR=internal/domain/$$LOWER; \
	if [ -d "$$DOMAIN_DIR" ]; then \
		echo "[ERROR] Directory $$DOMAIN_DIR already exists."; \
		exit 1; \
	fi; \
	mkdir -p $$DOMAIN_DIR; \
	echo "[INFO] Created directory: $$DOMAIN_DIR"; \
	DOMAIN_FILE=$$DOMAIN_DIR/domain.go; \
	printf "package $$LOWER\n" >> $$DOMAIN_FILE; \
	printf "\n" >> $$DOMAIN_FILE; \
	printf "type $${PASCAL}Domain interface{}\n" >> $$DOMAIN_FILE; \
	printf "\n" >> $$DOMAIN_FILE; \
	printf "type $${CAMEL}Domain struct{}\n" >> $$DOMAIN_FILE; \
	printf "\n" >> $$DOMAIN_FILE; \
	printf "func New$${PASCAL}Domain() $${PASCAL}Domain {\n" >> $$DOMAIN_FILE; \
	printf "\treturn &$${CAMEL}Domain{}\n" >> $$DOMAIN_FILE; \
	printf "}\n" >> $$DOMAIN_FILE; \
	echo "[INFO] Created domain file: $$DOMAIN_FILE"; \
	REGISTRY_FILE=internal/domain/registry.go; \
	if grep -q "\"prabogo/internal/domain/$$LOWER\"" "$$REGISTRY_FILE"; then \
		echo "[INFO] Import for $$LOWER already exists in $$REGISTRY_FILE"; \
	else \
		awk '/^import \($$/{print;print "\t\"prabogo/internal/domain/'"$$LOWER"'\"";next}1' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Added import for $$LOWER to $$REGISTRY_FILE"; \
	fi; \
	if grep -q "$${PASCAL}() $${LOWER}.$${PASCAL}Domain" "$$REGISTRY_FILE"; then \
		echo "[INFO] Interface method $${PASCAL}() already exists in $$REGISTRY_FILE"; \
	else \
		awk '/^type Domain interface \{$$/{print;print "\t'"$${PASCAL}"'() '"$${LOWER}"'.'"$${PASCAL}"'Domain";next}1' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Added $${PASCAL}() method to Domain interface in $$REGISTRY_FILE"; \
	fi; \
	if grep -q "func (d \*domain) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		echo "[INFO] Function $${PASCAL}() already exists in $$REGISTRY_FILE"; \
	else \
		printf "\n" >> $$REGISTRY_FILE; \
		printf "func (d *domain) $${PASCAL}() $${LOWER}.$${PASCAL}Domain {\n" >> $$REGISTRY_FILE; \
		printf "\treturn $${LOWER}.New$${PASCAL}Domain()\n" >> $$REGISTRY_FILE; \
		printf "}\n" >> $$REGISTRY_FILE; \
		echo "[INFO] Added $${PASCAL}() function to $$REGISTRY_FILE"; \
	fi; \
	echo "[INFO] Domain generation completed successfully!"

migration-postgres:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make migration postgres VAL=name"; \
		exit 1; \
	fi; \
	MIGRATION_DIR=internal/migration/postgres; \
	FILE_COUNT=$$(find $$MIGRATION_DIR -type f -name "*.go" | wc -l); \
	NEXT_NUM=$$((FILE_COUNT + 1)); \
	LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=$$MIGRATION_DIR/$${NEXT_NUM}_$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[ERROR] File $$DST already exists."; \
		exit 1; \
	fi; \
	printf "package migrations\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "import (\n" >> $$DST; \
	printf "\t\"context\"\n" >> $$DST; \
	printf "\t\"database/sql\"\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "\t\"github.com/pressly/goose/v3\"\n" >> $$DST; \
	printf ")\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "func init() {\n" >> $$DST; \
	printf "\tgoose.AddMigrationContext(up$${PASCAL}, down$${PASCAL})\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "func up$${PASCAL}(ctx context.Context, tx *sql.Tx) error {\n" >> $$DST; \
	printf "\t// This code is executed when the migration is applied.\n" >> $$DST; \
	printf "\t_, err := tx.Exec(\`CREATE TABLE IF NOT EXISTS $${LOWER}s (\n" >> $$DST; \
	printf "\t\tid SERIAL PRIMARY KEY,\n" >> $$DST; \
	printf "\t\tcreated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,\n" >> $$DST; \
	printf "\t\tupdated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL\n" >> $$DST; \
	printf "\t);\`)\n" >> $$DST; \
	printf "\tif err != nil {\n" >> $$DST; \
	printf "\t\treturn err\n" >> $$DST; \
	printf "\t}\n" >> $$DST; \
	printf "\treturn nil\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	printf "\n" >> $$DST; \
	printf "func down$${PASCAL}(ctx context.Context, tx *sql.Tx) error {\n" >> $$DST; \
	printf "\t// This code is executed when the migration is rolled back.\n" >> $$DST; \
	printf "\t_, err := tx.Exec(\`DROP TABLE $${LOWER}s;\`)\n" >> $$DST; \
	printf "\tif err != nil {\n" >> $$DST; \
	printf "\t\treturn err\n" >> $$DST; \
	printf "\t}\n" >> $$DST; \
	printf "\treturn nil\n" >> $$DST; \
	printf "}\n" >> $$DST; \
	echo "[INFO] Created migration file: $$DST"

inbound-http-fiber:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make inbound-http-fiber VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/inbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}HttpPort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}HttpPort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}HttpPort interface to $$DST"; \
    else \
      echo "[INFO] $${PASCAL}HttpPort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package inbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}HttpPort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with HTTP interface"; \
	fi; \
	FIBER_ADAPTER_DST=internal/adapter/inbound/fiber/$${LOWER}.go; \
	if [ -f "$$FIBER_ADAPTER_DST" ]; then \
		echo "[INFO] Fiber adapter file $$FIBER_ADAPTER_DST already exists."; \
	else \
		printf "package fiber_inbound_adapter\n" >> $$FIBER_ADAPTER_DST; \
		printf "\n" >> $$FIBER_ADAPTER_DST; \
		printf "import (\n" >> $$FIBER_ADAPTER_DST; \
		printf "\t\"prabogo/internal/domain\"\n" >> $$FIBER_ADAPTER_DST; \
		printf "\tinbound_port \"prabogo/internal/port/inbound\"\n" >> $$FIBER_ADAPTER_DST; \
		printf ")\n" >> $$FIBER_ADAPTER_DST; \
		printf "\n" >> $$FIBER_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {\n" >> $$FIBER_ADAPTER_DST; \
		printf "\tdomain domain.Domain\n" >> $$FIBER_ADAPTER_DST; \
		printf "}\n" >> $$FIBER_ADAPTER_DST; \
		printf "\n" >> $$FIBER_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter(\n" >> $$FIBER_ADAPTER_DST; \
		printf "\tdomain domain.Domain,\n" >> $$FIBER_ADAPTER_DST; \
		printf ") inbound_port.$${PASCAL}HttpPort {\n" >> $$FIBER_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{\n" >> $$FIBER_ADAPTER_DST; \
		printf "\t\tdomain: domain,\n" >> $$FIBER_ADAPTER_DST; \
		printf "\t}\n" >> $$FIBER_ADAPTER_DST; \
		printf "}\n" >> $$FIBER_ADAPTER_DST; \
		echo "[INFO] Created fiber adapter file: $$FIBER_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/inbound/fiber/registry.go; \
	if ! grep -q "func (s \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		METHOD_TEXT="\nfunc (s *adapter) $${PASCAL}() inbound_port.$${PASCAL}HttpPort {\n\treturn New$${PASCAL}Adapter(s.domain)\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in fiber adapter registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/inbound/registry_http.go; \
	if grep -q "type HttpPort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}HttpPort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}HttpPort" '/type HttpPort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated HttpPort interface in port registry"; \
		else \
			echo "[INFO] $${PASCAL} method already exists in HttpPort interface"; \
		fi; \
	else \
		echo "[ERROR] HttpPort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi;

inbound-message-rabbitmq:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make inbound-message-rabbitmq VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/inbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}MessagePort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}MessagePort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}MessagePort interface to $$DST"; \
		else \
			echo "[INFO] $${PASCAL}MessagePort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package inbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}MessagePort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with Message interface"; \
	fi; \
	RABBITMQ_ADAPTER_DST=internal/adapter/inbound/rabbitmq/$${LOWER}.go; \
	if [ -f "$$RABBITMQ_ADAPTER_DST" ]; then \
		echo "[INFO] RabbitMQ adapter file $$RABBITMQ_ADAPTER_DST already exists."; \
	else \
		printf "package rabbitmq_inbound_adapter\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "import (\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\t\"prabogo/internal/domain\"\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\tinbound_port \"prabogo/internal/port/inbound\"\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf ")\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\tdomain domain.Domain\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "}\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter(\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\tdomain domain.Domain,\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf ") inbound_port.$${PASCAL}MessagePort {\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\t\tdomain: domain,\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\t}\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "}\n" >> $$RABBITMQ_ADAPTER_DST; \
		echo "[INFO] Created RabbitMQ adapter file: $$RABBITMQ_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/inbound/rabbitmq/registry.go; \
	if ! grep -q "func (a \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		echo "[INFO] Adding $${PASCAL} method to registry adapter..."; \
		METHOD_TEXT="\nfunc (a *adapter) $${PASCAL}() inbound_port.$${PASCAL}MessagePort {\n\treturn New$${PASCAL}Adapter(a.domain)\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in rabbitmq registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/inbound/registry_message.go; \
	if grep -q "type MessagePort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}MessagePort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}MessagePort" '/type MessagePort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated MessagePort interface in port registry"; \
		else \
			echo "[INFO] $${PASCAL}MessagePort method already exists in MessagePort interface"; \
		fi; \
	else \
		echo "[ERROR] MessagePort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi;

inbound-command:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make inbound-command VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/inbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}CommandPort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}CommandPort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}CommandPort interface to $$DST"; \
		else \
			echo "[INFO] $${PASCAL}CommandPort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package inbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}CommandPort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with Command interface"; \
	fi; \
	COMMAND_ADAPTER_DST=internal/adapter/inbound/command/$${LOWER}.go; \
	if [ -f "$$COMMAND_ADAPTER_DST" ]; then \
		echo "[INFO] Command adapter file $$COMMAND_ADAPTER_DST already exists."; \
	else \
		printf "package command_inbound_adapter\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\n" >> $$COMMAND_ADAPTER_DST; \
		printf "import (\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\t\"prabogo/internal/domain\"\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\tinbound_port \"prabogo/internal/port/inbound\"\n" >> $$COMMAND_ADAPTER_DST; \
		printf ")\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\n" >> $$COMMAND_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\tdomain domain.Domain\n" >> $$COMMAND_ADAPTER_DST; \
		printf "}\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\n" >> $$COMMAND_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter(\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\tdomain domain.Domain,\n" >> $$COMMAND_ADAPTER_DST; \
		printf ") inbound_port.$${PASCAL}CommandPort {\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\t\tdomain: domain,\n" >> $$COMMAND_ADAPTER_DST; \
		printf "\t}\n" >> $$COMMAND_ADAPTER_DST; \
		printf "}\n" >> $$COMMAND_ADAPTER_DST; \
		echo "[INFO] Created Command adapter file: $$COMMAND_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/inbound/command/registry.go; \
	if ! grep -q "func (s \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		echo "[INFO] Adding $${PASCAL} method to registry adapter..."; \
		METHOD_TEXT="\nfunc (s *adapter) $${PASCAL}() inbound_port.$${PASCAL}CommandPort {\n\treturn New$${PASCAL}Adapter(s.domain)\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in command registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/inbound/registry_command.go; \
	if grep -q "type CommandPort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}CommandPort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}CommandPort" '/type CommandPort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated CommandPort interface in registry"; \
		else \
			echo "[INFO] $${PASCAL}CommandPort method already exists in CommandPort interface"; \
		fi; \
	else \
		echo "[ERROR] CommandPort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi;

outbound-database-postgres:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make outbound-database-postgres VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/outbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}DatabasePort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}DatabasePort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}DatabasePort interface to $$DST"; \
	else \
	  echo "[INFO] $${PASCAL}DatabasePort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package outbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}DatabasePort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with Database interface"; \
	fi; \
	POSTGRES_ADAPTER_DST=internal/adapter/outbound/postgres/$${LOWER}.go; \
	if [ -f "$$POSTGRES_ADAPTER_DST" ]; then \
		echo "[INFO] Postgres adapter file $$POSTGRES_ADAPTER_DST already exists."; \
	else \
		printf "package postgres_outbound_adapter\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "import (\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\toutbound_port \"prabogo/internal/port/outbound\"\n" >> $$POSTGRES_ADAPTER_DST; \
		printf ")\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "const table$${PASCAL} = \"$${LOWER}s\"\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\tdb outbound_port.DatabaseExecutor\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "}\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter(\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\tdb outbound_port.DatabaseExecutor,\n" >> $$POSTGRES_ADAPTER_DST; \
		printf ") outbound_port.$${PASCAL}DatabasePort {\n" >> $$POSTGRES_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{\n" >> $$POSTGRES_ADAPTER_DST; \
	printf "\t\tdb: db,\n" >> $$POSTGRES_ADAPTER_DST; \
	printf "\t}\n" >> $$POSTGRES_ADAPTER_DST; \
	printf "}\n" >> $$POSTGRES_ADAPTER_DST; \
	echo "[INFO] Created postgres adapter file: $$POSTGRES_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/outbound/postgres/registry.go; \
	if ! grep -q "func (s \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		METHOD_TEXT="\nfunc (s *adapter) $${PASCAL}() outbound_port.$${PASCAL}DatabasePort {\n\tif s.dbexecutor != nil {\n\t\treturn New$${PASCAL}Adapter(s.dbexecutor)\n\t}\n\treturn New$${PASCAL}Adapter(s.db)\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in postgres registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/outbound/registry_database.go; \
	if grep -q "type DatabasePort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}DatabasePort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}DatabasePort" '/type DatabasePort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated DatabasePort interface in registry"; \
		else \
			echo "[INFO] $${PASCAL} method already exists in DatabasePort interface"; \
		fi; \
	else \
		echo "[ERROR] DatabasePort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi; \
	go generate ./internal/port/outbound/registry_database.go; \
	echo "[INFO] Successfully generated mock for outbound DatabasePort."

outbound-http:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make outbound-http VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/outbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}HttpPort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}HttpPort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}HttpPort interface to $$DST"; \
	else \
	  echo "[INFO] $${PASCAL}HttpPort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package outbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}HttpPort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with HTTP interface"; \
	fi; \
	HTTP_ADAPTER_DST=internal/adapter/outbound/http/$${LOWER}.go; \
	if [ -f "$$HTTP_ADAPTER_DST" ]; then \
		echo "[INFO] HTTP adapter file $$HTTP_ADAPTER_DST already exists."; \
	else \
		printf "package http_outbound_adapter\n" >> $$HTTP_ADAPTER_DST; \
		printf "\n" >> $$HTTP_ADAPTER_DST; \
		printf "import (\n" >> $$HTTP_ADAPTER_DST; \
		printf "\toutbound_port \"prabogo/internal/port/outbound\"\n" >> $$HTTP_ADAPTER_DST; \
		printf ")\n" >> $$HTTP_ADAPTER_DST; \
		printf "\n" >> $$HTTP_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {}\n" >> $$HTTP_ADAPTER_DST; \
		printf "\n" >> $$HTTP_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter() outbound_port.$${PASCAL}HttpPort {\n" >> $$HTTP_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{}\n" >> $$HTTP_ADAPTER_DST; \
		printf "}\n" >> $$HTTP_ADAPTER_DST; \
		echo "[INFO] Created http adapter file: $$HTTP_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/outbound/http/registry.go; \
	if ! grep -q "func (s \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		echo "[INFO] Adding $${PASCAL} method to registry adapter..."; \
		METHOD_TEXT="\nfunc (s *adapter) $${PASCAL}() outbound_port.$${PASCAL}HttpPort {\n\treturn New$${PASCAL}Adapter()\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in command registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/outbound/registry_http.go; \
	if grep -q "type HttpPort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}HttpPort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}HttpPort" '/type HttpPort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated HttpPort interface in registry"; \
		else \
			echo "[INFO] $${PASCAL} method already exists in HttpPort interface"; \
		fi; \
	else \
		echo "[ERROR] HttpPort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi; \
	go generate ./internal/port/outbound/registry_http.go; \
	echo "[INFO] Successfully generated mock for outbound HttpPort."

outbound-message-rabbitmq:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make outbound-message-rabbitmq VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/outbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}MessagePort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}MessagePort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}MessagePort interface to $$DST"; \
		else \
			echo "[INFO] $${PASCAL}MessagePort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package outbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}MessagePort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with Message interface"; \
	fi; \
	RABBITMQ_ADAPTER_DST=internal/adapter/outbound/rabbitmq/$${LOWER}.go; \
	if [ -f "$$RABBITMQ_ADAPTER_DST" ]; then \
		echo "[INFO] RabbitMQ adapter file $$RABBITMQ_ADAPTER_DST already exists."; \
	else \
		printf "package rabbitmq_outbound_adapter\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "import (\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\toutbound_port \"prabogo/internal/port/outbound\"\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf ")\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {}\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter() outbound_port.$${PASCAL}MessagePort {\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{}\n" >> $$RABBITMQ_ADAPTER_DST; \
		printf "}\n" >> $$RABBITMQ_ADAPTER_DST; \
		echo "[INFO] Created rabbitmq adapter file: $$RABBITMQ_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/outbound/rabbitmq/registry.go; \
	if ! grep -q "func (s \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		echo "[INFO] Adding $${PASCAL} method to registry adapter..."; \
		METHOD_TEXT="\nfunc (s *adapter) $${PASCAL}() outbound_port.$${PASCAL}MessagePort {\n\treturn New$${PASCAL}Adapter()\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in rabbitmq registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/outbound/registry_message.go; \
	if grep -q "type MessagePort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}MessagePort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}MessagePort" '/type MessagePort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated MessagePort interface in registry"; \
		else \
			echo "[INFO] $${PASCAL} method already exists in MessagePort interface"; \
		fi; \
	else \
		echo "[ERROR] MessagePort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi; \
	go generate ./internal/port/outbound/registry_message.go; \
	echo "[INFO] Successfully generated mock for outbound MessagePort."

outbound-cache-redis:
	@if [ -z "$(VAL)" ]; then \
		echo "[ERROR] Please provide VAL, e.g. make outbound-cache-redis VAL=name"; \
		exit 1; \
	fi
	@LOWER=$$(echo $(VAL) | tr '[:upper:]' '[:lower:]'); \
	if [[ "$$LOWER" == *_* ]]; then \
		CAMEL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {$$1=$$1; for(i=2;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
		PASCAL=$$(echo "$$LOWER" | awk 'BEGIN{FS="_";OFS=""} {for(i=1;i<=NF;i++) $$i=toupper(substr($$i,1,1)) substr($$i,2)} 1'); \
	else \
		CAMEL=$$(echo $$LOWER); \
		PASCAL=$$(echo $$LOWER | awk '{print toupper(substr($$0,1,1)) substr($$0,2)}'); \
	fi; \
	DST=internal/port/outbound/$${LOWER}.go; \
	if [ -f "$$DST" ]; then \
		echo "[INFO] File $$DST already exists."; \
		if ! grep -q "$${PASCAL}CachePort" "$$DST"; then \
			printf "\n" >> $$DST; \
			printf "type $${PASCAL}CachePort interface {}\n" >> $$DST; \
			echo "[INFO] Added $${PASCAL}CachePort interface to $$DST"; \
		else \
			echo "[INFO] $${PASCAL}CachePort interface already exists in $$DST"; \
		fi; \
	else \
		printf "package outbound_port\n" >> $$DST; \
		printf "\n" >> $$DST; \
		printf "type $${PASCAL}CachePort interface {}\n" >> $$DST; \
		echo "[INFO] Created port interface file: $$DST with Cache interface"; \
	fi; \
	REDIS_ADAPTER_DST=internal/adapter/outbound/redis/$${LOWER}.go; \
	if [ -f "$$REDIS_ADAPTER_DST" ]; then \
		echo "[INFO] Redis adapter file $$REDIS_ADAPTER_DST already exists."; \
	else \
		printf "package redis_outbound_adapter\n" >> $$REDIS_ADAPTER_DST; \
		printf "\n" >> $$REDIS_ADAPTER_DST; \
		printf "import (\n" >> $$REDIS_ADAPTER_DST; \
		printf "\toutbound_port \"prabogo/internal/port/outbound\"\n" >> $$REDIS_ADAPTER_DST; \
		printf ")\n" >> $$REDIS_ADAPTER_DST; \
		printf "\n" >> $$REDIS_ADAPTER_DST; \
		printf "type $${CAMEL}Adapter struct {}\n" >> $$REDIS_ADAPTER_DST; \
		printf "\n" >> $$REDIS_ADAPTER_DST; \
		printf "func New$${PASCAL}Adapter() outbound_port.$${PASCAL}CachePort {\n" >> $$REDIS_ADAPTER_DST; \
		printf "\treturn &$${CAMEL}Adapter{}\n" >> $$REDIS_ADAPTER_DST; \
		printf "}\n" >> $$REDIS_ADAPTER_DST; \
		echo "[INFO] Created redis adapter file: $$REDIS_ADAPTER_DST"; \
	fi; \
	REGISTRY_FILE=internal/adapter/outbound/redis/registry.go; \
	if ! grep -q "func (s \*adapter) $${PASCAL}()" "$$REGISTRY_FILE"; then \
		echo "[INFO] Adding $${PASCAL} method to registry adapter..."; \
		METHOD_TEXT="\nfunc (s *adapter) $${PASCAL}() outbound_port.$${PASCAL}CachePort {\n\treturn New$${PASCAL}Adapter()\n}"; \
		awk -v m="$$METHOD_TEXT" '1; END{print m}' "$$REGISTRY_FILE" > "$$REGISTRY_FILE.tmp" && mv "$$REGISTRY_FILE.tmp" "$$REGISTRY_FILE"; \
		echo "[INFO] Appended $${PASCAL} method to the bottom of $$REGISTRY_FILE"; \
	else \
		echo "[INFO] $${PASCAL} method already exists in redis registry"; \
	fi; \
	REGISTRY_INTERFACE_FILE=internal/port/outbound/registry_cache.go; \
	if grep -q "type CachePort interface" "$$REGISTRY_INTERFACE_FILE"; then \
		if ! grep -q "$${PASCAL}() $${PASCAL}CachePort" "$$REGISTRY_INTERFACE_FILE"; then \
			awk -v m="\t$${PASCAL}() $${PASCAL}CachePort" '/type CachePort interface *{/{print;print m;next}1' "$$REGISTRY_INTERFACE_FILE" > "$$REGISTRY_INTERFACE_FILE.tmp" && mv "$$REGISTRY_INTERFACE_FILE.tmp" "$$REGISTRY_INTERFACE_FILE"; \
			echo "[INFO] Updated CachePort interface in registry"; \
		else \
			echo "[INFO] $${PASCAL} method already exists in CachePort interface"; \
		fi; \
	else \
		echo "[ERROR] CachePort interface not found in $$REGISTRY_INTERFACE_FILE"; \
	fi; \
	go generate ./internal/port/outbound/registry_cache.go; \
	echo "[INFO] Successfully generated mock for outbound CachePort."

# Interactive target selector using fzf (if available) or basic shell selection
# This target displays an interactive menu to select and execute other Makefile targets
# It handles different parameter requirements based on the target type
# Works on macOS, Linux, and Windows (with WSL or Git Bash)
run:
	@if command -v fzf >/dev/null 2>&1; then \
		target=$$(grep -E "^[a-zA-Z0-9_-]+:" $(MAKEFILE_LIST) | grep -v "run:" | sed 's/:.*//' | sort | fzf --height=10 --prompt="Select Makefile target: "); \
	else \
		echo "[INFO] fzf not found, using basic selection menu"; \
		targets=$$(grep -E "^[a-zA-Z0-9_-]+:" $(MAKEFILE_LIST) | grep -v "run:" | sed 's/:.*//' | sort); \
		i=1; \
		for t in $$targets; do \
			echo "$$i) $$t"; \
			i=$$((i+1)); \
		done; \
		echo "Enter the number of the target to run:"; \
		read -r choice; \
		target=$$(echo "$$targets" | sed -n "$${choice}p"); \
		echo "[INFO] Selected: $$target"; \
	fi; \
	if [ -n "$$target" ]; then \
		echo "[INFO] Selected target: $$target"; \
		case "$$target" in \
			"model"|"domain"|"migration-postgres"|"inbound-http-fiber"|"inbound-message-rabbitmq"|"inbound-command"|"outbound-database-postgres"|"outbound-http"|"outbound-message-rabbitmq"|"outbound-cache-redis") \
				printf "Enter VAL parameter: "; \
				val=$$(bash -c 'read -r val && echo "$$val"'); \
				if [ -n "$$val" ]; then \
					make $$target VAL=$$val; \
				else \
					echo "[ERROR] VAL parameter is required for target: $$target"; \
				fi \
				;; \
			"message") \
				printf "Enter SUB parameter: "; \
				sub=$$(bash -c 'read -r sub && echo "$$sub"'); \
				printf "Force rebuild? (y/N): "; \
				build=$$(bash -c 'read -r build && echo "$$build"'); \
				if [ -n "$$sub" ]; then \
					if [ "$$build" = "y" ] || [ "$$build" = "Y" ]; then \
						make $$target SUB=$$sub BUILD=true; \
					else \
						make $$target SUB=$$sub; \
					fi \
				else \
					echo "[ERROR] SUB parameter is required for target: $$target"; \
				fi \
				;; \
			"command") \
				printf "Enter CMD parameter: "; \
				cmd=$$(bash -c 'read -r cmd && echo "$$cmd"'); \
				printf "Enter VAL parameter: "; \
				val=$$(bash -c 'read -r val && echo "$$val"'); \
				printf "Force rebuild? (y/N): "; \
				build=$$(bash -c 'read -r build && echo "$$build"'); \
				if [ -n "$$cmd" ] && [ -n "$$val" ]; then \
					if [ "$$build" = "y" ] || [ "$$build" = "Y" ]; then \
						make $$target CMD=$$cmd VAL=$$val BUILD=true; \
					else \
						make $$target CMD=$$cmd VAL=$$val; \
					fi \
				else \
					echo "[ERROR] Both CMD and VAL parameters are required for target: $$target"; \
				fi \
				;; \
			"http") \
				printf "Force rebuild? (y/N): "; \
				build=$$(bash -c 'read -r build && echo "$$build"'); \
				if [ "$$build" = "y" ] || [ "$$build" = "Y" ]; then \
					make $$target BUILD=true; \
				else \
					make $$target; \
				fi \
				;; \
			*) \
				echo "[INFO] Running target: $$target"; \
				make $$target; \
				;; \
		esac; \
	else \
		echo "[INFO] No target selected. Exiting."; \
	fi

generate-mocks:
	@echo "[INFO] Generating mocks from go:generate directives..."
	@go generate ./internal/port/outbound/registry_database.go
	@echo "[INFO] Successfully generated mock for outbound DatabasePort."
	@go generate ./internal/port/outbound/registry_http.go
	@echo "[INFO] Successfully generated mock for outbound HttpPort."
	@go generate ./internal/port/outbound/registry_cache.go
	@echo "[INFO] Successfully generated mock for outbound CachePort."
	@go generate ./internal/port/outbound/registry_message.go
	@echo "[INFO] Successfully generated mock for outbound MessagePort."

lint:
	@echo "[INFO] Running golangci-lint..."
	@golangci-lint run ./...

test:
	@echo "[INFO] Running unit tests..."
	@go test -v -race ./internal/domain/... ./internal/adapter/...

test-coverage:
	@echo "[INFO] Running tests with coverage..."
	@go test -v -race -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "[INFO] Coverage report generated: coverage.html"

test-integration:
	@echo "[INFO] Running integration tests..."
	@go test -v -tags=integration ./tests/integration/...