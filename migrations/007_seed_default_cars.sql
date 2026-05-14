-- +goose Up

INSERT INTO cars (brand, model, year, plate_number, daily_rate, seats, fuel, transmission, status, image_url)
VALUES
  ('Toyota', 'Camry', 2023, '01A777AA', 58.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1621007947382-bb3c3994e3fb?auto=format&fit=crop&w=1200&q=80'),
  ('BMW', 'X5', 2022, '01B505BB', 115.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1555215695-3004980ad54e?auto=format&fit=crop&w=1200&q=80'),
  ('Hyundai', 'Elantra', 2021, '01H221HA', 42.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1619767886558-efdc259cde1a?auto=format&fit=crop&w=1200&q=80'),
  ('Mercedes', 'C-Class', 2023, '01M909MA', 96.00, 5, 'Petrol', 'Automatic', 'maintenance', 'https://images.unsplash.com/photo-1618843479313-40f8afb4b4d8?auto=format&fit=crop&w=1200&q=80'),
  ('Chevrolet', 'Malibu', 2022, '01C222CH', 52.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1533473359331-0135ef1b58bf?auto=format&fit=crop&w=1200&q=80'),
  ('Kia', 'K5', 2024, '01K505KA', 67.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1609521263047-f8f205293f24?auto=format&fit=crop&w=1200&q=80'),
  ('Nissan', 'Altima', 2023, '01N404NA', 61.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1494976388531-d1058494cdd8?auto=format&fit=crop&w=1200&q=80'),
  ('Volkswagen', 'Jetta', 2021, '01V303VW', 45.00, 5, 'Petrol', 'Automatic', 'available', 'https://images.unsplash.com/photo-1503376780353-7e6692767b70?auto=format&fit=crop&w=1200&q=80')
ON CONFLICT (plate_number) DO UPDATE SET image_url = COALESCE(cars.image_url, EXCLUDED.image_url);
