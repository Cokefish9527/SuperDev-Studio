-- Migration SQL
-- Created: 2026-03-05T18:17:25.646684

-- Create analytics table
CREATE TABLE "analytics" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create experiences table
CREATE TABLE "experiences" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create operations table
CREATE TABLE "operations" (
    "id" UUID PRIMARY KEY,
    "name" VARCHAR(255),
    "status" VARCHAR(255),
    "created_at" TIMESTAMP,
    "updated_at" TIMESTAMP
);

-- Create searchs table
CREATE TABLE "searchs" (
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
