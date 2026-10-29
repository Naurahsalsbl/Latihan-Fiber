package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"latihan-fiber/modul4/app/model"
)

var (
	ErrPortfolioNotFound = errors.New("portofolio belum dibuat")
	ErrProjectNotFound   = errors.New("proyek tidak ditemukan")
	ErrProjectLimit      = errors.New("jumlah proyek sudah mencapai batas")
)

// MaxProjectsPerPortfolio membatasi pertumbuhan array projects. Satu dokumen
// MongoDB maksimal 16 MB; array yang dibiarkan tumbuh tanpa batas suatu saat
// akan menabrak batas itu.
const MaxProjectsPerPortfolio = 20

type PortfolioRepository interface {
	EnsureIndexes(ctx context.Context) error
	FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error)
	Replace(ctx context.Context, studentID int, p model.Portfolio) (model.Portfolio, error)
	AddProject(ctx context.Context, studentID int, p model.Project) (model.Project, error)
	DeleteProject(ctx context.Context, studentID int, projectID string) error
	TopTechnologies(ctx context.Context, limit int) ([]model.TechnologyCount, error)
}

// ---------- bentuk dokumen SEBAGAIMANA TERSIMPAN di MongoDB ----------

type portfolioDocument struct {
	ID           bson.ObjectID         `bson:"_id,omitempty"`
	StudentID    int                   `bson:"student_id"`
	Headline     string                `bson:"headline"`
	Bio          string                `bson:"bio"`
	Projects     []projectDocument     `bson:"projects"`
	Certificates []certificateDocument `bson:"certificates"`
	CreatedAt    time.Time             `bson:"created_at"`
	UpdatedAt    time.Time             `bson:"updated_at"`
}

type projectDocument struct {
	ID           bson.ObjectID `bson:"_id"`
	Title        string        `bson:"title"`
	Technologies []string      `bson:"technologies"`
	URL          string        `bson:"url"`
	Year         int           `bson:"year"`
}

type certificateDocument struct {
	ID     bson.ObjectID `bson:"_id"`
	Name   string        `bson:"name"`
	Issuer string        `bson:"issuer"`
	Date   time.Time     `bson:"date"`
}

func (d projectDocument) toModel() model.Project {
	techs := d.Technologies
	if techs == nil {
		techs = []string{}
	}
	return model.Project{
		ID: d.ID.Hex(), Title: d.Title, Technologies: techs,
		URL: d.URL, Year: d.Year,
	}
}

func newProjectDocument(p model.Project) projectDocument {
	techs := p.Technologies
	if techs == nil {
		techs = []string{}
	}
	return projectDocument{
		ID: bson.NewObjectID(), Title: p.Title, Technologies: techs,
		URL: p.URL, Year: p.Year,
	}
}

func (d portfolioDocument) toModel() model.Portfolio {
	projects := make([]model.Project, 0, len(d.Projects))
	for _, p := range d.Projects {
		projects = append(projects, p.toModel())
	}
	certs := make([]model.Certificate, 0, len(d.Certificates))
	for _, c := range d.Certificates {
		certs = append(certs, model.Certificate{
			ID: c.ID.Hex(), Name: c.Name, Issuer: c.Issuer, Date: c.Date,
		})
	}
	return model.Portfolio{
		ID: d.ID.Hex(), StudentID: d.StudentID,
		Headline: d.Headline, Bio: d.Bio,
		Projects: projects, Certificates: certs,
		CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type portfolioMongoRepository struct{ coll *mongo.Collection }

func NewPortfolioRepository(db *mongo.Database) PortfolioRepository {
	return &portfolioMongoRepository{coll: db.Collection("student_portfolios")}
}

// EnsureIndexes dijalankan sekali saat aplikasi menyala. Membuat index yang
// sudah ada tidak melakukan apa pun, sehingga aman dipanggil berulang kali.
func (r *portfolioMongoRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			// Satu mahasiswa hanya boleh punya SATU portofolio. Aturan ini
			// dijaga database, bukan hanya diperiksa di kode. Index ini juga
			// dipakai semua endpoint yang mencari berdasarkan student_id
			// (GET, PUT, POST proyek, DELETE proyek).
			Keys:    bson.D{{Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("student_id_unique"),
		},
	})
	if err != nil {
		return fmt.Errorf("membuat index student_portfolios: %w", err)
	}
	return nil
}

func (r *portfolioMongoRepository) FindByStudent(ctx context.Context, studentID int) (model.Portfolio, error) {
	var doc portfolioDocument
	err := r.coll.FindOne(ctx, bson.M{"student_id": studentID}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, fmt.Errorf("mengambil portofolio: %w", err)
	}
	return doc.toModel(), nil
}

// Replace membuat portofolio bila belum ada, atau menggantinya bila sudah ada
// (upsert). Seluruh isi projects dan certificates diganti; setiap elemen
// mendapat _id baru.
//
// Dua PUT pertama yang datang serentak untuk mahasiswa yang sama bisa sama-sama
// mencoba menyisipkan dokumen. Index unik menolak salah satunya dengan
// duplicate key; dokumen sudah ada pada saat itu, sehingga mencoba sekali lagi
// akan berhasil sebagai update biasa.
func (r *portfolioMongoRepository) Replace(ctx context.Context, studentID int, p model.Portfolio) (model.Portfolio, error) {
	t := mongoNow()

	projects := make([]projectDocument, 0, len(p.Projects))
	for _, pr := range p.Projects {
		projects = append(projects, newProjectDocument(pr))
	}
	certs := make([]certificateDocument, 0, len(p.Certificates))
	for _, c := range p.Certificates {
		certs = append(certs, certificateDocument{
			ID: bson.NewObjectID(), Name: c.Name, Issuer: c.Issuer, Date: c.Date,
		})
	}

	update := bson.M{
		"$set": bson.M{
			"headline":     p.Headline,
			"bio":          p.Bio,
			"projects":     projects,
			"certificates": certs,
			"updated_at":   t,
		},
		// created_at hanya diisi ketika dokumen baru dibuat.
		"$setOnInsert": bson.M{"created_at": t},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		var doc portfolioDocument
		err := r.coll.FindOneAndUpdate(ctx,
			bson.M{"student_id": studentID}, update, opts).Decode(&doc)
		if err == nil {
			return doc.toModel(), nil
		}
		if !mongo.IsDuplicateKeyError(err) {
			return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", err)
		}
		lastErr = err
	}
	return model.Portfolio{}, fmt.Errorf("menyimpan portofolio: %w", lastErr)
}

// AddProject menambahkan satu proyek ke dalam array projects, maksimal 20.
//
// Syarat "projects.19 tidak ada" berarti array belum berisi 20 elemen
// (indeks 0 sampai 19). Pemeriksaan batas dan $push terjadi dalam SATU
// operasi, sehingga sepuluh request serentak ketika tinggal dua tempat
// kosong hanya meloloskan tepat dua.
func (r *portfolioMongoRepository) AddProject(ctx context.Context, studentID int, p model.Project) (model.Project, error) {
	doc := newProjectDocument(p)
	limitField := "projects." + strconv.Itoa(MaxProjectsPerPortfolio-1)

	var updated portfolioDocument
	err := r.coll.FindOneAndUpdate(ctx,
		bson.M{"student_id": studentID, limitField: bson.M{"$exists": false}},
		bson.M{
			"$push": bson.M{"projects": doc},
			"$set":  bson.M{"updated_at": mongoNow()},
		},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&updated)
	if err == nil {
		for _, pr := range updated.Projects {
			if pr.ID == doc.ID {
				return pr.toModel(), nil
			}
		}
		return doc.toModel(), nil
	}
	if !errors.Is(err, mongo.ErrNoDocuments) {
		return model.Project{}, fmt.Errorf("menambah proyek: %w", err)
	}

	// Tidak ada dokumen yang cocok: portofolionya belum ada, atau sudah penuh.
	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return model.Project{}, fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return model.Project{}, ErrPortfolioNotFound
	}
	return model.Project{}, ErrProjectLimit
}

// DeleteProject menghapus satu proyek dengan $pull.
//
// project _id ikut ditaruh di filter. Dengan begitu MatchedCount == 0 berarti
// portofolionya tidak ada ATAU proyeknya tidak ada, dan keduanya dibedakan
// sesudahnya. Tanpa itu, $set updated_at membuat ModifiedCount selalu 1
// walaupun tidak ada proyek yang terhapus.
func (r *portfolioMongoRepository) DeleteProject(ctx context.Context, studentID int, projectID string) error {
	pid, err := parseObjectID(projectID)
	if err != nil {
		return err
	}

	res, err := r.coll.UpdateOne(ctx,
		bson.M{"student_id": studentID, "projects._id": pid},
		bson.M{
			"$pull": bson.M{"projects": bson.M{"_id": pid}},
			"$set":  bson.M{"updated_at": mongoNow()},
		},
	)
	if err != nil {
		return fmt.Errorf("menghapus proyek: %w", err)
	}
	if res.MatchedCount > 0 {
		return nil
	}

	count, err := r.coll.CountDocuments(ctx, bson.M{"student_id": studentID})
	if err != nil {
		return fmt.Errorf("memeriksa portofolio: %w", err)
	}
	if count == 0 {
		return ErrPortfolioNotFound
	}
	return ErrProjectNotFound
}

// TopTechnologies menghitung teknologi yang paling banyak dipakai pada seluruh
// portofolio. Dua kali $unwind: pertama memecah array projects menjadi satu
// dokumen per proyek, kedua memecah array technologies menjadi satu dokumen
// per teknologi. Barulah $group dapat menghitungnya.
func (r *portfolioMongoRepository) TopTechnologies(ctx context.Context, limit int) ([]model.TechnologyCount, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.M{
			"_id":   "$projects.technologies",
			"total": bson.M{"$sum": 1},
		}}},
		{{Key: "$sort", Value: bson.D{
			{Key: "total", Value: -1},
			{Key: "_id", Value: 1},
		}}},
		{{Key: "$limit", Value: int64(limit)}},
	}

	cursor, err := r.coll.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("menghitung statistik teknologi: %w", err)
	}
	defer cursor.Close(ctx)

	var rows []struct {
		Technology string `bson:"_id"`
		Total      int    `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, fmt.Errorf("membaca statistik teknologi: %w", err)
	}

	result := make([]model.TechnologyCount, 0, len(rows))
	for _, row := range rows {
		result = append(result, model.TechnologyCount{Technology: row.Technology, Total: row.Total})
	}
	return result, nil
}