package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			CREATE EXTENSION IF NOT EXISTS postgis;

			-- Roles table
			CREATE TABLE IF NOT EXISTS roles (
				id BIGSERIAL PRIMARY KEY,
				name VARCHAR(50) UNIQUE NOT NULL
			);

			INSERT INTO roles (name) VALUES ('CUSTOMER'), ('MERCHANT'), ('CREATOR'), ('ADMIN')
			ON CONFLICT (name) DO NOTHING;

			-- User Roles junction table
			CREATE TABLE IF NOT EXISTS user_roles (
				id BIGSERIAL PRIMARY KEY,
				user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
				UNIQUE(user_id, role_id)
			);

			-- Merchant Profiles table
			CREATE TABLE IF NOT EXISTS merchant_profiles (
				id BIGSERIAL PRIMARY KEY,
				owner_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				business_name VARCHAR(255) NOT NULL,
				business_description TEXT,
				business_type VARCHAR(50) NOT NULL DEFAULT 'RETAIL',
				verification_status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_merchant_profiles_owner ON merchant_profiles(owner_user_id);

			-- Merchant Members table
			CREATE TABLE IF NOT EXISTS merchant_members (
				id BIGSERIAL PRIMARY KEY,
				merchant_id BIGINT NOT NULL REFERENCES merchant_profiles(id) ON DELETE CASCADE,
				user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				permission VARCHAR(50) NOT NULL DEFAULT 'MANAGER',
				status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(merchant_id, user_id)
			);

			-- Shops table with PostGIS location
			CREATE TABLE IF NOT EXISTS shops (
				id BIGSERIAL PRIMARY KEY,
				merchant_id BIGINT NOT NULL REFERENCES merchant_profiles(id) ON DELETE CASCADE,
				name VARCHAR(255) NOT NULL,
				description TEXT,
				phone VARCHAR(50),
				address TEXT NOT NULL,
				city VARCHAR(100),
				state VARCHAR(100),
				pincode VARCHAR(20),
				country VARCHAR(100) DEFAULT 'India',
				location GEOGRAPHY(POINT, 4326),
				status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
				verification_status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_shops_location ON shops USING GIST(location);
			CREATE INDEX IF NOT EXISTS idx_shops_merchant ON shops(merchant_id);

			-- Products central catalogue table
			CREATE TABLE IF NOT EXISTS products (
				id BIGSERIAL PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				brand VARCHAR(255),
				description TEXT,
				category_id BIGINT,
				specifications JSONB,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);

			-- Shop Products listing table
			CREATE TABLE IF NOT EXISTS shop_products (
				id BIGSERIAL PRIMARY KEY,
				shop_id BIGINT NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
				product_id BIGINT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
				price NUMERIC(12, 2) NOT NULL,
				currency VARCHAR(10) NOT NULL DEFAULT 'INR',
				availability VARCHAR(50) NOT NULL DEFAULT 'AVAILABLE',
				stock_updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				UNIQUE(shop_id, product_id)
			);
			CREATE INDEX IF NOT EXISTS idx_shop_products_shop ON shop_products(shop_id);
			CREATE INDEX IF NOT EXISTS idx_shop_products_product ON shop_products(product_id);

			-- Customer Saved Locations table
			CREATE TABLE IF NOT EXISTS user_saved_locations (
				id BIGSERIAL PRIMARY KEY,
				user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
				label VARCHAR(100),
				address TEXT,
				city VARCHAR(100),
				pincode VARCHAR(20),
				location GEOGRAPHY(POINT, 4326),
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			CREATE INDEX IF NOT EXISTS idx_user_saved_locations_user ON user_saved_locations(user_id);
			CREATE INDEX IF NOT EXISTS idx_user_saved_locations_geo ON user_saved_locations USING GIST(location);

			-- Add relationship columns to video_uploads
			ALTER TABLE video_uploads
			ADD COLUMN IF NOT EXISTS merchant_id BIGINT REFERENCES merchant_profiles(id) ON DELETE SET NULL,
			ADD COLUMN IF NOT EXISTS shop_id BIGINT REFERENCES shops(id) ON DELETE SET NULL,
			ADD COLUMN IF NOT EXISTS product_id BIGINT REFERENCES products(id) ON DELETE SET NULL,
			ADD COLUMN IF NOT EXISTS shop_product_id BIGINT REFERENCES shop_products(id) ON DELETE SET NULL;
		`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`
			ALTER TABLE video_uploads
			DROP COLUMN IF EXISTS merchant_id,
			DROP COLUMN IF EXISTS shop_id,
			DROP COLUMN IF EXISTS product_id,
			DROP COLUMN IF EXISTS shop_product_id;

			DROP TABLE IF EXISTS user_saved_locations;
			DROP TABLE IF EXISTS shop_products;
			DROP TABLE IF EXISTS products;
			DROP TABLE IF EXISTS shops;
			DROP TABLE IF EXISTS merchant_members;
			DROP TABLE IF EXISTS merchant_profiles;
			DROP TABLE IF EXISTS user_roles;
			DROP TABLE IF EXISTS roles;
		`)
		return err
	})
}
