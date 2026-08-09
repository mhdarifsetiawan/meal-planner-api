-- Create master_ingredients table
CREATE TABLE IF NOT EXISTS master_ingredients (
    id SERIAL PRIMARY KEY,
    category VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL UNIQUE,
    default_unit VARCHAR(50) NOT NULL DEFAULT 'kg',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create ingredient_aliases table
CREATE TABLE IF NOT EXISTS ingredient_aliases (
    id SERIAL PRIMARY KEY,
    master_ingredient_id INT NOT NULL REFERENCES master_ingredients(id) ON DELETE CASCADE,
    alias_name VARCHAR(255) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for fast lookup by category and lowercase names
CREATE INDEX IF NOT EXISTS idx_master_ingredients_category ON master_ingredients(category);
CREATE INDEX IF NOT EXISTS idx_ingredient_aliases_alias_name ON ingredient_aliases(LOWER(alias_name));

-- Seed initial master ingredients
INSERT INTO master_ingredients (id, category, name, default_unit) VALUES
-- Cabai
(1, 'Cabai', 'Cabai Rawit Merah', 'kg'),
(2, 'Cabai', 'Cabai Rawit Hijau', 'kg'),
(3, 'Cabai', 'Cabai Merah Keriting', 'kg'),
(4, 'Cabai', 'Cabai Merah Besar', 'kg'),
(5, 'Cabai', 'Cabai Hijau Besar', 'kg'),

-- Bawang
(6, 'Bawang', 'Bawang Merah', 'kg'),
(7, 'Bawang', 'Bawang Putih', 'kg'),
(8, 'Bawang', 'Bawang Bombay', 'kg'),
(9, 'Bawang', 'Daun Bawang', 'ikat'),

-- Daging & Protein
(10, 'Daging & Protein', 'Daging Ayam', 'kg'),
(11, 'Daging & Protein', 'Daging Sapi', 'kg'),
(12, 'Daging & Protein', 'Telur Ayam', 'kg'),
(13, 'Daging & Protein', 'Tahu Putih', 'papan'),
(14, 'Daging & Protein', 'Tempe', 'papan'),
(15, 'Daging & Protein', 'Ikan Gurame', 'kg'),
(16, 'Daging & Protein', 'Udang', 'kg'),

-- Beras & Sembako
(17, 'Beras & Sembako', 'Beras Medium', 'kg'),
(18, 'Beras & Sembako', 'Beras Premium', 'kg'),
(19, 'Beras & Sembako', 'Minyak Goreng', 'liter'),
(20, 'Beras & Sembako', 'Gula Pasir', 'kg'),

-- Sayuran
(21, 'Sayuran', 'Kangkung', 'ikat'),
(22, 'Sayuran', 'Bayam', 'ikat'),
(23, 'Sayuran', 'Wortel', 'kg'),
(24, 'Sayuran', 'Kentang', 'kg'),
(25, 'Sayuran', 'Tomat', 'kg')
ON CONFLICT (name) DO NOTHING;

-- Set sequence next value
SELECT setval('master_ingredients_id_seq', (SELECT MAX(id) FROM master_ingredients));

-- Seed initial ingredient aliases
INSERT INTO ingredient_aliases (master_ingredient_id, alias_name) VALUES
-- Cabai Rawit Merah
(1, 'cabe rawit'),
(1, 'cabe rawit merah'),
(1, 'lombok rawit'),
(1, 'cabe sret'),
(1, 'cabai setan'),

-- Cabai Rawit Hijau
(2, 'cabe rawit hijau'),
(2, 'cabe lalap'),
(2, 'cabe hijau kecil'),

-- Cabai Merah Keriting
(3, 'cabe keriting'),
(3, 'cabe merah keriting'),
(3, 'cabai keriting'),

-- Cabai Merah Besar
(4, 'cabe merah besar'),
(4, 'cabe merah'),
(4, 'cabai merah'),

-- Bawang Merah
(6, 'brambang'),
(6, 'bawang merah lokal'),

-- Bawang Putih
(7, 'bawang putih kating'),

-- Daging Ayam
(10, 'ayam potong'),
(10, 'daging ayam ras'),
(10, 'ayam'),

-- Daging Sapi
(11, 'daging sapi lokal'),
(11, 'daging sapi segar'),

-- Telur Ayam
(12, 'telur ayam ras'),
(12, 'telur'),

-- Minyak Goreng
(19, 'minyak goreng kemasan'),
(19, 'minyak curah')
ON CONFLICT (alias_name) DO NOTHING;
