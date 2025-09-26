-- Create tables for testing

-- 1. Old EAV model
CREATE TABLE users (
    user_id BIGSERIAL PRIMARY KEY
);

CREATE TABLE user_attributes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(user_id),
    key VARCHAR(50),
    value TEXT
);

CREATE INDEX idx_user_attrs_user_id ON user_attributes(user_id);
CREATE INDEX idx_user_attrs_key ON user_attributes(key);

-- 2. New denormalized model
CREATE TABLE user_profiles (
    user_id BIGINT PRIMARY KEY,
    country VARCHAR(2),
    tier VARCHAR(20),
    last_active_at TIMESTAMP DEFAULT NOW(),
    has_purchased BOOLEAN DEFAULT FALSE,
    total_spend DECIMAL(10,2) DEFAULT 0
) PARTITION BY HASH (user_id);

-- Create 10 partitions
CREATE TABLE user_profiles_0 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 0);
CREATE TABLE user_profiles_1 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 1);
CREATE TABLE user_profiles_2 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 2);
CREATE TABLE user_profiles_3 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 3);
CREATE TABLE user_profiles_4 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 4);
CREATE TABLE user_profiles_5 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 5);
CREATE TABLE user_profiles_6 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 6);
CREATE TABLE user_profiles_7 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 7);
CREATE TABLE user_profiles_8 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 8);
CREATE TABLE user_profiles_9 PARTITION OF user_profiles FOR VALUES WITH (modulus 10, remainder 9);

-- Optimal indexes
CREATE INDEX idx_country ON user_profiles USING btree (country);
CREATE INDEX idx_tier ON user_profiles USING btree (tier);
CREATE INDEX idx_active_recent ON user_profiles USING BRIN (last_active_at);
CREATE INDEX idx_has_purchased ON user_profiles USING btree (has_purchased) WHERE has_purchased = true;
CREATE INDEX idx_high_spender ON user_profiles USING btree (total_spend) WHERE total_spend > 100;

-- 3. Predicate cache
CREATE TABLE predicate_cache (
    predicate_hash VARCHAR(64) PRIMARY KEY,
    user_count INT,
    last_updated TIMESTAMP DEFAULT NOW()
);

-- Function to populate test data
CREATE OR REPLACE FUNCTION populate_test_data(num_users INT)
RETURNS void AS $$
DECLARE
    i INT;
    countries TEXT[] := ARRAY['US', 'UK', 'DE', 'FR', 'JP', 'AU', 'CA', 'BR', 'IN'];
    tiers TEXT[] := ARRAY['free', 'free', 'free', 'free', 'gold', 'platinum'];
BEGIN
    -- Populate old EAV model
    INSERT INTO users (user_id) SELECT generate_series(1, num_users);

    -- Add attributes for each user
    FOR i IN 1..num_users LOOP
        -- country attribute (40% US)
        IF random() < 0.4 THEN
            INSERT INTO user_attributes (user_id, key, value) VALUES (i, 'country', 'US');
        ELSE
            INSERT INTO user_attributes (user_id, key, value)
            VALUES (i, 'country', countries[1 + floor(random() * 9)::int]);
        END IF;

        -- tier attribute
        INSERT INTO user_attributes (user_id, key, value)
        VALUES (i, 'tier', tiers[1 + floor(random() * 6)::int]);

        -- last_active attribute
        INSERT INTO user_attributes (user_id, key, value)
        VALUES (i, 'last_active_at', (NOW() - (random() * INTERVAL '365 days'))::text);

        -- has_purchased attribute
        INSERT INTO user_attributes (user_id, key, value)
        VALUES (i, 'has_purchased', (random() < 0.2)::text);

        -- total_spend attribute
        INSERT INTO user_attributes (user_id, key, value)
        VALUES (i, 'total_spend', (random() * 1000)::text);
    END LOOP;

    -- Populate new denormalized model
    INSERT INTO user_profiles (user_id, country, tier, last_active_at, has_purchased, total_spend)
    SELECT
        u.user_id,
        MAX(CASE WHEN ua.key = 'country' THEN ua.value END) as country,
        MAX(CASE WHEN ua.key = 'tier' THEN ua.value END) as tier,
        MAX(CASE WHEN ua.key = 'last_active_at' THEN ua.value::timestamp END) as last_active_at,
        bool_or(CASE WHEN ua.key = 'has_purchased' THEN ua.value::boolean END) as has_purchased,
        MAX(CASE WHEN ua.key = 'total_spend' THEN ua.value::decimal END) as total_spend
    FROM users u
    JOIN user_attributes ua ON u.user_id = ua.user_id
    GROUP BY u.user_id;

    -- Analyze tables for query optimization
    ANALYZE user_attributes;
    ANALYZE user_profiles;
END;
$$ LANGUAGE plpgsql;

-- Populate 100k users for initial test
SELECT populate_test_data(100000);