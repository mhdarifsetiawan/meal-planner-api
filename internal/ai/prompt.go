package ai

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt returns the system prompt instructing AI on JSON format and constraints.
// catalog: list of canonical ingredient names from master_ingredients + aliases to guide AI naming.
func BuildSystemPrompt(catalog []string) string {
	catalogSection := ""
	if len(catalog) > 0 {
		catalogSection = `
REFERENSI NAMA BAHAN (WAJIB DIIKUTI):
Gunakan HANYA nama bahan dari daftar berikut ini. Pilih nama yang PALING SPESIFIK dan TEPAT sesuai kebutuhan resep.
Jika bahan yang dibutuhkan tidak ada di daftar, boleh menggunakan nama yang paling mendekati.
Daftar nama resmi bahan:
` + "- " + strings.Join(catalog, "\n- ") + "\n"
	}

	return `Anda adalah ahli gizi dan koki profesional Indonesia.
Tugas Anda adalah merekomendasikan 3 variasi menu masak harian khas Indonesia dalam bentuk JSON yang terstruktur.

ATURAN STRICT HARGA & OPSI:
1. Kembalikan HANYA format JSON valid tanpa teks markdown pembuka/penutup.
2. Setiap estimasi harga bahan dan total harga HARUS dalam integer Rupiah. Estimasi harga bahan adalah HARGA PORSI YANG DIBUTUHKAN UNTUK MEMASAK REAKSI TERSEBUT (contoh: 2 butir telur = Rp 4.000, BUKAN Rp 28.000 per kg).
3. Sesuaikan dengan batasan pantangan/alergi dan target budget user.` + catalogSection + `
3 OPSI MENU HARUS MENGIKUTI STRUKTUR & URUTAN INI:
- Opsi 1 (Index 0): "Opsi Hemat" -> Total harga HARUS PALING MURAH dari ketiga opsi dan HARUS <= target budget.
- Opsi 2 (Index 1): "Opsi Bergizi" -> Menu tinggi protein dan nutrisi lengkap seimbang.
- Opsi 3 (Index 2): "Opsi Praktis" -> Menu sederhana dengan waktu masak cepat (< 20 menit).

PENTING: Urutkan array "options" sehingga Opsi 1 (Index 0) memiliki total estimasi harga TERKECIL dibanding Opsi 2 dan Opsi 3!

FORMAT JSON OUTPUT YANG WAJIB DIIKUTI:
{
  "options": [
    {
      "recipe_name": "Nama Masakan",
      "description": "Deskripsi singkat masakan dan alasan nutrisinya",
      "estimated_total_price": 35000,
      "goal_tags": ["hemat", "sehat"],
      "ingredients": [
        {
          "name": "Nama Bahan Utama",
          "quantity": "2",
          "unit": "butir",
          "estimated_price": 4000
        }
      ]
    }
  ]
}`
}


// BuildUserPrompt formats user parameters into a prompt for the AI model.
func BuildUserPrompt(params MenuGenerateParams) string {
	restrictionsStr := "Tidak ada"
	if len(params.Restrictions) > 0 {
		restrictionsStr = strings.Join(params.Restrictions, ", ")
	}

	prompt := fmt.Sprintf(`Target Goal: %s
Budget Amount: Rp %d
Budget Period: %s
Jumlah Anggota Keluarga: %d orang
Pantangan / Alergi: %s`,
		params.Goal,
		params.BudgetAmount,
		params.BudgetPeriod,
		params.HouseholdSize,
		restrictionsStr,
	)

	if len(params.ExcludeRecipes) > 0 {
		excludeStr := strings.Join(params.ExcludeRecipes, ", ")
		prompt += fmt.Sprintf("\n\n⚠️ SANGAT PENTING (DILARANG DUPLIKASI):\nJangan menyarankan kembali resep-resep berikut karena sudah pernah disarankan sebelumnya: %s.\nBerikan 3 pilihan resep rekomendasi baru yang 100%% berbeda dari daftar tersebut!", excludeStr)
	}

	prompt += "\n\nBerikan 3 opsi menu rekomendasi masakan harian Indonesia sesuai parameter di atas!"

	return prompt
}
