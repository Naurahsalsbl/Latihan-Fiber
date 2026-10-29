package service

import "strings"

// NormalizeTechnologies merapikan daftar teknologi sebelum disimpan: huruf
// kecil, tanpa spasi di tepi, tanpa duplikat, urutan pertama dipertahankan.
//
// Tanpa normalisasi, "Go", "go", dan " GO " tercatat sebagai tiga teknologi
// berbeda, dan statistik teknologi terpopuler menjadi tidak bermakna.
// Fungsi ini murni: tidak menyentuh database maupun Fiber, sehingga dapat
// diuji hanya dengan memanggilnya.
func NormalizeTechnologies(techs []string) []string {
	result := make([]string, 0, len(techs))
	seen := make(map[string]struct{}, len(techs))
	for _, t := range techs {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, dup := seen[t]; dup {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	return result
}