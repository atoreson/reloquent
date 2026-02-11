-- =============================================================================
-- Reloquent Trial Mode: Pagila Schema (Simplified)
-- Based on the Pagila sample database for PostgreSQL
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Lookup / reference tables
-- ---------------------------------------------------------------------------

CREATE TABLE actor (
    actor_id    SERIAL PRIMARY KEY,
    first_name  VARCHAR(45) NOT NULL,
    last_name   VARCHAR(45) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE category (
    category_id SERIAL PRIMARY KEY,
    name        VARCHAR(25) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE language (
    language_id SERIAL PRIMARY KEY,
    name        VARCHAR(20) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

-- ---------------------------------------------------------------------------
-- Geography
-- ---------------------------------------------------------------------------

CREATE TABLE country (
    country_id  SERIAL PRIMARY KEY,
    country     VARCHAR(50) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE city (
    city_id     SERIAL PRIMARY KEY,
    city        VARCHAR(50) NOT NULL,
    country_id  INT NOT NULL REFERENCES country (country_id),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_city_country_id ON city (country_id);

CREATE TABLE address (
    address_id  SERIAL PRIMARY KEY,
    address     VARCHAR(50) NOT NULL,
    address2    VARCHAR(50),
    district    VARCHAR(20) NOT NULL,
    city_id     INT NOT NULL REFERENCES city (city_id),
    postal_code VARCHAR(10),
    phone       VARCHAR(20) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_address_city_id ON address (city_id);

-- ---------------------------------------------------------------------------
-- Store & staff
-- ---------------------------------------------------------------------------

CREATE TABLE store (
    store_id    SERIAL PRIMARY KEY,
    address_id  INT NOT NULL REFERENCES address (address_id),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_store_address_id ON store (address_id);

CREATE TABLE staff (
    staff_id    SERIAL PRIMARY KEY,
    first_name  VARCHAR(45) NOT NULL,
    last_name   VARCHAR(45) NOT NULL,
    address_id  INT NOT NULL REFERENCES address (address_id),
    email       VARCHAR(50),
    store_id    INT NOT NULL REFERENCES store (store_id),
    active      BOOLEAN NOT NULL DEFAULT true,
    username    VARCHAR(16) NOT NULL,
    password    VARCHAR(40),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_staff_address_id ON staff (address_id);
CREATE INDEX idx_staff_store_id ON staff (store_id);

-- ---------------------------------------------------------------------------
-- Film catalog
-- ---------------------------------------------------------------------------

CREATE TABLE film (
    film_id          SERIAL PRIMARY KEY,
    title            VARCHAR(255) NOT NULL,
    description      TEXT,
    release_year     INT,
    language_id      INT NOT NULL REFERENCES language (language_id),
    rental_duration  SMALLINT NOT NULL DEFAULT 3,
    rental_rate      NUMERIC(4,2) NOT NULL DEFAULT 4.99,
    length           SMALLINT,
    replacement_cost NUMERIC(5,2) NOT NULL DEFAULT 19.99,
    rating           VARCHAR(10) DEFAULT 'G',
    special_features TEXT[],
    last_update      TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_film_language_id ON film (language_id);
CREATE INDEX idx_film_title ON film (title);

CREATE TABLE film_actor (
    actor_id    INT NOT NULL REFERENCES actor (actor_id),
    film_id     INT NOT NULL REFERENCES film (film_id),
    last_update TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, film_id)
);

CREATE INDEX idx_film_actor_film_id ON film_actor (film_id);

CREATE TABLE film_category (
    film_id     INT NOT NULL REFERENCES film (film_id),
    category_id INT NOT NULL REFERENCES category (category_id),
    last_update TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (film_id, category_id)
);

CREATE INDEX idx_film_category_category_id ON film_category (category_id);

-- ---------------------------------------------------------------------------
-- Inventory
-- ---------------------------------------------------------------------------

CREATE TABLE inventory (
    inventory_id SERIAL PRIMARY KEY,
    film_id      INT NOT NULL REFERENCES film (film_id),
    store_id     INT NOT NULL REFERENCES store (store_id),
    last_update  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_inventory_film_id ON inventory (film_id);
CREATE INDEX idx_inventory_store_id ON inventory (store_id);

-- ---------------------------------------------------------------------------
-- Customer
-- ---------------------------------------------------------------------------

CREATE TABLE customer (
    customer_id SERIAL PRIMARY KEY,
    store_id    INT NOT NULL REFERENCES store (store_id),
    first_name  VARCHAR(45) NOT NULL,
    last_name   VARCHAR(45) NOT NULL,
    email       VARCHAR(50),
    address_id  INT NOT NULL REFERENCES address (address_id),
    activebool  BOOLEAN NOT NULL DEFAULT true,
    create_date DATE NOT NULL DEFAULT CURRENT_DATE,
    last_update TIMESTAMP DEFAULT now(),
    active      INT
);

CREATE INDEX idx_customer_store_id ON customer (store_id);
CREATE INDEX idx_customer_address_id ON customer (address_id);
CREATE INDEX idx_customer_last_name ON customer (last_name);

-- ---------------------------------------------------------------------------
-- Rental & payment
-- ---------------------------------------------------------------------------

CREATE TABLE rental (
    rental_id    SERIAL PRIMARY KEY,
    rental_date  TIMESTAMP NOT NULL,
    inventory_id INT NOT NULL REFERENCES inventory (inventory_id),
    customer_id  INT NOT NULL REFERENCES customer (customer_id),
    return_date  TIMESTAMP,
    staff_id     INT NOT NULL REFERENCES staff (staff_id),
    last_update  TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_rental_inventory_id ON rental (inventory_id);
CREATE INDEX idx_rental_customer_id ON rental (customer_id);
CREATE INDEX idx_rental_staff_id ON rental (staff_id);
CREATE INDEX idx_rental_date ON rental (rental_date);

CREATE TABLE payment (
    payment_id   SERIAL PRIMARY KEY,
    customer_id  INT NOT NULL REFERENCES customer (customer_id),
    staff_id     INT NOT NULL REFERENCES staff (staff_id),
    rental_id    INT NOT NULL REFERENCES rental (rental_id),
    amount       NUMERIC(5,2) NOT NULL,
    payment_date TIMESTAMP NOT NULL
);

CREATE INDEX idx_payment_customer_id ON payment (customer_id);
CREATE INDEX idx_payment_staff_id ON payment (staff_id);
CREATE INDEX idx_payment_rental_id ON payment (rental_id);
