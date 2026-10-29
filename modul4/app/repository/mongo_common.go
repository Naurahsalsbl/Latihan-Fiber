package repository

import (
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// ErrInvalidID dipakai bila id dari client bukan ObjectID yang sah.
//
// CATATAN: modul juga mendefinisikan ErrInvalidID di review_repository.go.
// Bila Anda menyalin repository ulasan dari modul, HAPUS satu dari dua
// definisi ini (dan juga fungsi parseID/mongoNow bila namanya bentrok).
var ErrInvalidID = errors.New("format id tidak valid")

// mongoNow membulatkan waktu ke milidetik. Tipe date pada BSON hanya
// menyimpan sampai milidetik, sedangkan time.Time di Go sampai nanodetik.
// Tanpa pembulatan, nilai pada response saat menyimpan akan berbeda dari
// nilai yang dibaca kembali dari database.
func mongoNow() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }

// parseObjectID mengubah string heksadesimal menjadi bson.ObjectID.
func parseObjectID(id string) (bson.ObjectID, error) {
	oid, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return bson.ObjectID{}, ErrInvalidID
	}
	return oid, nil
}