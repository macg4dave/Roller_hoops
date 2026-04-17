FROM postgres:16-alpine

COPY docker/dev/dev-seed.sql /seed/dev-seed.sql
