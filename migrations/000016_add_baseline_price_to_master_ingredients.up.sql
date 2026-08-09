-- Add baseline_price column to master_ingredients table
ALTER TABLE master_ingredients ADD COLUMN IF NOT EXISTS baseline_price INT NOT NULL DEFAULT 10000;

-- Seed realistic initial baseline prices (in Rupiah)
UPDATE master_ingredients SET baseline_price = 40000 WHERE name = 'Cabai Rawit Merah';
UPDATE master_ingredients SET baseline_price = 35000 WHERE name = 'Cabai Rawit Hijau';
UPDATE master_ingredients SET baseline_price = 38000 WHERE name = 'Cabai Merah Keriting';
UPDATE master_ingredients SET baseline_price = 35000 WHERE name = 'Cabai Merah Besar';
UPDATE master_ingredients SET baseline_price = 30000 WHERE name = 'Cabai Hijau Besar';

UPDATE master_ingredients SET baseline_price = 32000 WHERE name = 'Bawang Merah';
UPDATE master_ingredients SET baseline_price = 36000 WHERE name = 'Bawang Putih';
UPDATE master_ingredients SET baseline_price = 28000 WHERE name = 'Bawang Bombay';
UPDATE master_ingredients SET baseline_price = 15000 WHERE name = 'Daun Bawang';

UPDATE master_ingredients SET baseline_price = 35000 WHERE name = 'Daging Ayam';
UPDATE master_ingredients SET baseline_price = 130000 WHERE name = 'Daging Sapi';
UPDATE master_ingredients SET baseline_price = 28000 WHERE name = 'Telur Ayam';
UPDATE master_ingredients SET baseline_price = 5000 WHERE name = 'Tahu Putih';
UPDATE master_ingredients SET baseline_price = 6000 WHERE name = 'Tempe';
UPDATE master_ingredients SET baseline_price = 45000 WHERE name = 'Ikan Gurame';
UPDATE master_ingredients SET baseline_price = 80000 WHERE name = 'Udang';

UPDATE master_ingredients SET baseline_price = 15000 WHERE name = 'Beras Medium';
UPDATE master_ingredients SET baseline_price = 17500 WHERE name = 'Beras Premium';
UPDATE master_ingredients SET baseline_price = 18000 WHERE name = 'Minyak Goreng';
UPDATE master_ingredients SET baseline_price = 17000 WHERE name = 'Gula Pasir';

UPDATE master_ingredients SET baseline_price = 4000 WHERE name = 'Kangkung';
UPDATE master_ingredients SET baseline_price = 4000 WHERE name = 'Bayam';
UPDATE master_ingredients SET baseline_price = 12000 WHERE name = 'Wortel';
UPDATE master_ingredients SET baseline_price = 18000 WHERE name = 'Kentang';
UPDATE master_ingredients SET baseline_price = 14000 WHERE name = 'Tomat';
