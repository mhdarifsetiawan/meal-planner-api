-- Migration 001: Region tables (provinces & cities)

CREATE TABLE provinces (
    id   SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL
);

CREATE TABLE cities (
    id          SERIAL PRIMARY KEY,
    province_id INT NOT NULL REFERENCES provinces(id),
    name        VARCHAR(100) NOT NULL
);
