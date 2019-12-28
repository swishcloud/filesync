#!/bin/bash
#set up and run database for hydra
docker run \
 --name filesync-postgres \
 --rm \
 -p 5420:5432 \
 -e POSTGRES_USER=postgres \
 -e POSTGRES_PASSWORD=secret  \
 -e POSTGRES_DB=filesync \
 -d \
 postgres
echo sleep 5 seconds for waiting brand-new database to run
sleep 5