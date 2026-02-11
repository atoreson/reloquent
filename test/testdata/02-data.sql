-- Minimal Pagila data for integration tests
INSERT INTO actor (first_name, last_name) VALUES
('PENELOPE', 'GUINESS'), ('NICK', 'WAHLBERG'), ('ED', 'CHASE'),
('JENNIFER', 'DAVIS'), ('JOHNNY', 'LOLLOBRIGIDA'), ('BETTE', 'NICHOLSON'),
('GRACE', 'MOSTEL'), ('MATTHEW', 'JOHANSSON'), ('JOE', 'SWANK'),
('CHRISTIAN', 'GABLE'), ('ZERO', 'CAGE'), ('KARL', 'BERRY');

INSERT INTO category (name) VALUES
('Action'), ('Animation'), ('Children'), ('Classics'),
('Comedy'), ('Documentary'), ('Drama'), ('Family'),
('Foreign'), ('Games'), ('Horror'), ('Music'),
('New'), ('Sci-Fi'), ('Sports'), ('Travel');

INSERT INTO language (name) VALUES
('English'), ('Italian'), ('Japanese'), ('Mandarin'), ('French'), ('German');

INSERT INTO country (country) VALUES
('United States'), ('Canada'), ('United Kingdom'), ('Australia'),
('Japan'), ('Germany'), ('France'), ('India'), ('Brazil'), ('Mexico');

INSERT INTO city (city, country_id) VALUES
('New York', 1), ('Los Angeles', 1), ('Toronto', 2), ('London', 3),
('Sydney', 4), ('Tokyo', 5), ('Berlin', 6), ('Paris', 7),
('Mumbai', 8), ('Sao Paulo', 9), ('Mexico City', 10), ('Chicago', 1);

INSERT INTO address (address, district, city_id, postal_code, phone) VALUES
('123 Main St', 'Manhattan', 1, '10001', '555-0101'),
('456 Oak Ave', 'Hollywood', 2, '90028', '555-0102'),
('789 Maple Dr', 'Downtown', 3, 'M5V 2T6', '555-0103'),
('10 Baker St', 'Westminster', 4, 'NW1 6XE', '555-0104'),
('55 George St', 'CBD', 5, '2000', '555-0105'),
('1 Shibuya', 'Shibuya', 6, '150-0002', '555-0106'),
('22 Unter den Linden', 'Mitte', 7, '10117', '555-0107'),
('8 Rue de Rivoli', 'Paris 1er', 8, '75001', '555-0108');

INSERT INTO store (address_id) VALUES (1), (2);

INSERT INTO staff (first_name, last_name, address_id, email, store_id, username, password) VALUES
('Mike', 'Hillyer', 3, 'Mike.Hillyer@sakilastaff.com', 1, 'Mike', 'changeme'),
('Jon', 'Stephens', 4, 'Jon.Stephens@sakilastaff.com', 2, 'Jon', 'changeme');

INSERT INTO film (title, description, release_year, language_id, rental_duration, rental_rate, length, replacement_cost, rating) VALUES
('ACADEMY DINOSAUR', 'An epic drama of a feminist and a mad scientist', 2006, 1, 6, 0.99, 86, 20.99, 'PG'),
('ACE GOLDFINGER', 'A astounding epistle of a database administrator', 2006, 1, 3, 4.99, 48, 12.99, 'G'),
('ADAPTATION HOLES', 'A astounding reflection of a lumberjack and a car', 2006, 1, 7, 2.99, 50, 18.99, 'NC-17'),
('AFFAIR PREJUDICE', 'A fanciful documentary of a frisbee and a lumberjack', 2006, 1, 5, 2.99, 117, 26.99, 'G'),
('AFRICAN EGG', 'A fast-paced documentary of a pastry chef', 2006, 1, 6, 2.99, 130, 22.99, 'G'),
('AGENT TRUMAN', 'A intrepid panorama of a robot and a boy', 2006, 1, 3, 2.99, 169, 17.99, 'PG'),
('AIRPLANE SIERRA', 'A touching saga of a hunter and an explorer', 2006, 1, 6, 4.99, 62, 28.99, 'PG-13'),
('AIRPORT POLLOCK', 'A epic tale of a moose and a girl', 2006, 1, 6, 4.99, 54, 15.99, 'R'),
('ALABAMA DEVIL', 'A thoughtful panorama of a database administrator', 2006, 1, 3, 2.99, 114, 21.99, 'PG-13'),
('ALADDIN CALENDAR', 'A action-packed tale of a man and a lumberjack', 2006, 1, 6, 4.99, 63, 24.99, 'NC-17');

INSERT INTO film_actor (actor_id, film_id) VALUES
(1, 1), (1, 2), (2, 3), (2, 4), (3, 5), (3, 6),
(4, 7), (4, 8), (5, 9), (5, 10), (6, 1), (6, 3),
(7, 2), (7, 5), (8, 4), (8, 7), (9, 6), (9, 8),
(10, 9), (10, 1), (11, 2), (11, 10), (12, 3), (12, 5);

INSERT INTO film_category (film_id, category_id) VALUES
(1, 6), (2, 11), (3, 6), (4, 11), (5, 8),
(6, 9), (7, 5), (8, 11), (9, 11), (10, 15);

INSERT INTO inventory (film_id, store_id) VALUES
(1, 1), (1, 1), (1, 2), (2, 2), (2, 2),
(3, 1), (4, 1), (4, 2), (5, 2), (6, 1),
(7, 1), (7, 2), (8, 2), (9, 1), (10, 2);

INSERT INTO customer (store_id, first_name, last_name, email, address_id) VALUES
(1, 'MARY', 'SMITH', 'MARY.SMITH@sakilacustomer.org', 5),
(1, 'PATRICIA', 'JOHNSON', 'PATRICIA.JOHNSON@sakilacustomer.org', 6),
(1, 'LINDA', 'WILLIAMS', 'LINDA.WILLIAMS@sakilacustomer.org', 7),
(2, 'BARBARA', 'JONES', 'BARBARA.JONES@sakilacustomer.org', 8),
(2, 'ELIZABETH', 'BROWN', 'ELIZABETH.BROWN@sakilacustomer.org', 5);

INSERT INTO rental (rental_date, inventory_id, customer_id, return_date, staff_id) VALUES
('2005-05-24 22:54:33', 1, 1, '2005-05-26 22:04:30', 1),
('2005-05-24 23:03:39', 2, 1, '2005-05-28 19:40:33', 1),
('2005-05-25 00:02:21', 3, 2, '2005-05-28 00:22:00', 2),
('2005-05-25 00:09:02', 4, 2, '2005-05-29 00:27:57', 1),
('2005-05-25 01:48:41', 5, 3, '2005-05-28 00:00:12', 2),
('2005-05-25 02:09:02', 6, 3, '2005-05-29 00:27:57', 1),
('2005-05-25 03:03:12', 7, 4, '2005-05-28 03:33:33', 2),
('2005-05-25 03:09:02', 8, 4, '2005-05-29 03:27:57', 1),
('2005-05-25 04:48:41', 9, 5, '2005-05-28 04:00:12', 2),
('2005-05-25 05:09:02', 10, 5, '2005-05-29 05:27:57', 1);

INSERT INTO payment (customer_id, staff_id, rental_id, amount, payment_date) VALUES
(1, 1, 1, 2.99, '2005-05-25 11:30:37'),
(1, 1, 2, 0.99, '2005-05-28 10:35:23'),
(2, 2, 3, 5.99, '2005-05-27 00:09:24'),
(2, 1, 4, 0.99, '2005-05-29 08:31:03'),
(3, 2, 5, 9.99, '2005-05-29 08:07:57'),
(3, 1, 6, 4.99, '2005-05-30 14:22:42'),
(4, 2, 7, 2.99, '2005-05-29 12:44:21'),
(4, 1, 8, 3.99, '2005-05-30 15:33:11'),
(5, 2, 9, 1.99, '2005-05-29 16:55:09'),
(5, 1, 10, 6.99, '2005-05-30 17:22:33');
