-- Create the test database
-- The main 'swopcart' database is created automatically via POSTGRES_DB env var
CREATE DATABASE swopcart_test;

-- Grant all privileges on the test database to the swopcart user
GRANT ALL PRIVILEGES ON DATABASE swopcart_test TO swopcart;
