package ai

import (
	"fmt"
	"strings"
)

// BuildSystemPrompt returns the system prompt instructing AI on JSON format and constraints.
func BuildSystemPrompt() string {
	return `Anda adalah ahli gizi dan koki profesional Indonesia.
Tugas Anda adalah merekomendasikan 3 variasi menu masak harian khas Indonesia dalam bentuk JSON yang terstruktur.

ATURAN STRICT:
1. Kembalikan HANYA format JSON valid tanpa teks markdown pembuka/penutup.
2. Setiap estimasi harga bahan dan total harga HARUS dalam integer Rupiah (tanpa desimal/koma).
3. Sesuaikan dengan batasan pantangan/alergi dan target budget user.

FORMAT JSON OUTPUT YANG WAJIB DIIKUTI:
{
  "options": [
    {
      "recipe_name": "Nama Masakan",
      "description": "Deskripsi singkat masakan dan alasan nutrisinya",
      "estimated_total_price": 45000,
      "goal_tags": ["hemat", "sehat"],
      "ingredients": [
        {
          "name": "Nama Bahan Utama",
          "quantity": "2",
          "unit": "potong",
          "estimated_price": 5000
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

	return fmt.Sprintf(`Target Goal: %s
Budget Amount: Rp %d
Budget Period: %s
Jumlah Anggota Keluarga: %d orang
Pantangan / Alergi: %s

Berikan 3 opsi menu rekomendasi masakan harian Indonesia sesuai parameter di atas!`,
		params.Goal,
		params.BudgetAmount,
		params.BudgetPeriod,
		params.HouseholdSize,
		restrictionsStr,
	)
}
