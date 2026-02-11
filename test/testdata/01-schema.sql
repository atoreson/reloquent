-- Minimal Pagila schema for integration tests
CREATE TABLE actor (
    actor_id SERIAL PRIMARY KEY,
    first_name VARCHAR(45) NOT NULL,
    last_name VARCHAR(45) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE category (
    category_id SERIAL PRIMARY KEY,
    name VARCHAR(25) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE language (
    language_id SERIAL PRIMARY KEY,
    name VARCHAR(20) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE country (
    country_id SERIAL PRIMARY KEY,
    country VARCHAR(50) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE city (
    city_id SERIAL PRIMARY KEY,
    city VARCHAR(50) NOT NULL,
    country_id INTEGER NOT NULL REFERENCES country(country_id),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE address (
    address_id SERIAL PRIMARY KEY,
    address VARCHAR(50) NOT NULL,
    address2 VARCHAR(50),
    district VARCHAR(20) NOT NULL,
    city_id INTEGER NOT NULL REFERENCES city(city_id),
    postal_code VARCHAR(10),
    phone VARCHAR(20) NOT NULL,
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE store (
    store_id SERIAL PRIMARY KEY,
    address_id INTEGER NOT NULL REFERENCES address(address_id),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE staff (
    staff_id SERIAL PRIMARY KEY,
    first_name VARCHAR(45) NOT NULL,
    last_name VARCHAR(45) NOT NULL,
    address_id INTEGER NOT NULL REFERENCES address(address_id),
    email VARCHAR(50),
    store_id INTEGER NOT NULL REFERENCES store(store_id),
    active BOOLEAN NOT NULL DEFAULT true,
    username VARCHAR(16) NOT NULL,
    password VARCHAR(40),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE film (
    film_id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    release_year INTEGER,
    language_id INTEGER NOT NULL REFERENCES language(language_id),
    rental_duration SMALLINT NOT NULL DEFAULT 3,
    rental_rate NUMERIC(4,2) NOT NULL DEFAULT 4.99,
    length SMALLINT,
    replacement_cost NUMERIC(5,2) NOT NULL DEFAULT 19.99,
    rating VARCHAR(10) DEFAULT 'G',
    special_features TEXT[],
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE film_actor (
    actor_id INTEGER NOT NULL REFERENCES actor(actor_id),
    film_id INTEGER NOT NULL REFERENCES film(film_id),
    last_update TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (actor_id, film_id)
);

CREATE TABLE film_category (
    film_id INTEGER NOT NULL REFERENCES film(film_id),
    category_id INTEGER NOT NULL REFERENCES category(category_id),
    last_update TIMESTAMP NOT NULL DEFAULT now(),
    PRIMARY KEY (film_id, category_id)
);

CREATE TABLE inventory (
    inventory_id SERIAL PRIMARY KEY,
    film_id INTEGER NOT NULL REFERENCES film(film_id),
    store_id INTEGER NOT NULL REFERENCES store(store_id),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE customer (
    customer_id SERIAL PRIMARY KEY,
    store_id INTEGER NOT NULL REFERENCES store(store_id),
    first_name VARCHAR(45) NOT NULL,
    last_name VARCHAR(45) NOT NULL,
    email VARCHAR(50),
    address_id INTEGER NOT NULL REFERENCES address(address_id),
    activebool BOOLEAN NOT NULL DEFAULT true,
    create_date DATE NOT NULL DEFAULT CURRENT_DATE,
    last_update TIMESTAMP DEFAULT now(),
    active INTEGER
);

CREATE TABLE rental (
    rental_id SERIAL PRIMARY KEY,
    rental_date TIMESTAMP NOT NULL,
    inventory_id INTEGER NOT NULL REFERENCES inventory(inventory_id),
    customer_id INTEGER NOT NULL REFERENCES customer(customer_id),
    return_date TIMESTAMP,
    staff_id INTEGER NOT NULL REFERENCES staff(staff_id),
    last_update TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE payment (
    payment_id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customer(customer_id),
    staff_id INTEGER NOT NULL REFERENCES staff(staff_id),
    rental_id INTEGER NOT NULL REFERENCES rental(rental_id),
    amount NUMERIC(5,2) NOT NULL,
    payment_date TIMESTAMP NOT NULL
);

-- Indexes on FK columns
CREATE INDEX idx_city_country ON city(country_id);
CREATE INDEX idx_address_city ON address(city_id);
CREATE INDEX idx_store_address ON store(address_id);
CREATE INDEX idx_staff_address ON staff(address_id);
CREATE INDEX idx_staff_store ON staff(store_id);
CREATE INDEX idx_film_language ON film(language_id);
CREATE INDEX idx_inventory_film ON inventory(film_id);
CREATE INDEX idx_inventory_store ON inventory(store_id);
CREATE INDEX idx_customer_store ON customer(store_id);
CREATE INDEX idx_customer_address ON customer(address_id);
CREATE INDEX idx_rental_inventory ON rental(inventory_id);
CREATE INDEX idx_rental_customer ON rental(customer_id);
CREATE INDEX idx_rental_staff ON rental(staff_id);
CREATE INDEX idx_payment_customer ON payment(customer_id);
CREATE INDEX idx_payment_staff ON payment(staff_id);
CREATE INDEX idx_payment_rental ON payment(rental_id);
