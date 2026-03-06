-- Migration SQL
-- Created: 2026-03-05T17:59:39.488597

-- Create analytics table
CREATE TABLE "analytics" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create contents table
CREATE TABLE "contents" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create notifications table
CREATE TABLE "notifications" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create cores table
CREATE TABLE "cores" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create workflows table
CREATE TABLE "workflows" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);
