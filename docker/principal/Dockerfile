FROM postgres:16

COPY schema.sql /docker-entrypoint-initdb.d/01-schema.sql
COPY seed.sql  /docker-entrypoint-initdb.d/02-seed.sql

EXPOSE 5432
