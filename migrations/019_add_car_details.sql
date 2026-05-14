-- +goose Up

ALTER TABLE cars ADD COLUMN IF NOT EXISTS seats INT;
ALTER TABLE cars ADD COLUMN IF NOT EXISTS fuel VARCHAR(32);
ALTER TABLE cars ADD COLUMN IF NOT EXISTS transmission VARCHAR(32);

UPDATE cars SET seats = 5 WHERE seats IS NULL;
UPDATE cars SET fuel = 'Petrol' WHERE fuel IS NULL OR fuel = '';
UPDATE cars SET transmission = 'Automatic' WHERE transmission IS NULL OR transmission = '';

ALTER TABLE cars ALTER COLUMN seats SET DEFAULT 5;
ALTER TABLE cars ALTER COLUMN fuel SET DEFAULT 'Petrol';
ALTER TABLE cars ALTER COLUMN transmission SET DEFAULT 'Automatic';

ALTER TABLE cars ALTER COLUMN seats SET NOT NULL;
ALTER TABLE cars ALTER COLUMN fuel SET NOT NULL;
ALTER TABLE cars ALTER COLUMN transmission SET NOT NULL;

ALTER TABLE cars DROP CONSTRAINT IF EXISTS cars_seats_check;
ALTER TABLE cars DROP CONSTRAINT IF EXISTS cars_fuel_check;
ALTER TABLE cars DROP CONSTRAINT IF EXISTS cars_transmission_check;

ALTER TABLE cars ADD CONSTRAINT cars_seats_check CHECK (seats BETWEEN 1 AND 12);
ALTER TABLE cars ADD CONSTRAINT cars_fuel_check CHECK (fuel <> '');
ALTER TABLE cars ADD CONSTRAINT cars_transmission_check CHECK (transmission <> '');
