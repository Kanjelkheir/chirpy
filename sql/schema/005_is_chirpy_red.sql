-- +goose Up
alter table users add column is_chirpy_red boolean default false;
