-- Migration 012: Seed major Indonesian provinces and cities

INSERT INTO provinces (id, name) VALUES
(1, 'DKI Jakarta'),
(2, 'Jawa Barat'),
(3, 'Jawa Tengah'),
(4, 'DI Yogyakarta'),
(5, 'Jawa Timur'),
(6, 'Banten'),
(7, 'Bali'),
(8, 'Sumatera Utara'),
(9, 'Sulawesi Selatan')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name;

INSERT INTO cities (id, province_id, name) VALUES
(1, 1, 'Jakarta Selatan'),
(2, 1, 'Jakarta Pusat'),
(3, 1, 'Jakarta Barat'),
(4, 1, 'Jakarta Timur'),
(5, 1, 'Jakarta Utara'),
(6, 2, 'Bandung'),
(7, 2, 'Bogor'),
(8, 2, 'Depok'),
(9, 2, 'Bekasi'),
(10, 6, 'Tangerang'),
(11, 6, 'Tangerang Selatan'),
(12, 3, 'Semarang'),
(13, 3, 'Surakarta (Solo)'),
(14, 4, 'Yogyakarta'),
(15, 5, 'Surabaya'),
(16, 5, 'Malang'),
(17, 7, 'Denpasar'),
(18, 8, 'Medan'),
(19, 9, 'Makassar')
ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, province_id = EXCLUDED.province_id;

-- Reset serial sequence so new inserts don't collide
SELECT setval('provinces_id_seq', (SELECT MAX(id) FROM provinces));
SELECT setval('cities_id_seq', (SELECT MAX(id) FROM cities));
