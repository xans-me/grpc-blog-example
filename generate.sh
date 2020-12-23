#!/bin/bash

protoc protobuff/*.proto --go_out=plugins=grpc:. --go_opt=paths=source_relative
