################################################################
# 
# Copyright 2026 Sudhakar Narayanamurthy. All rights reserved.
# Licensed under the Apache License, Version 2.0 (the "License")
# 
# Makefile convenience targets for compose build, up, down, and health checks.
#

.PHONY: build up down logs test-health

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

test-health:
	@curl -sf http://localhost:8080/health && echo " global OK"
	@curl -sf http://localhost:8081/health && echo " storage-tier-1 OK"
	@curl -sf http://localhost:8082/health && echo " storage-tier-2 OK"
