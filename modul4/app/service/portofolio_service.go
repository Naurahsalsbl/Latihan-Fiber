package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"latihan-fiber/modul4/app/model"
	"latihan-fiber/modul4/app/repository"
	"latihan-fiber/modul4/helper"
)

// PortfolioService memakai DUA database sekaligus: data mahasiswa dibaca dari
// PostgreSQL (lewat StudentRepository), portofolionya disimpan di MongoDB.
// Service-lah satu-satunya tempat yang mengenal keduanya.
type PortfolioService struct {
	portfolios repository.PortfolioRepository
	students   repository.StudentRepository
	perms      *helper.PermissionSet
}

func NewPortfolioService(
	portfolios repository.PortfolioRepository,
	students repository.StudentRepository,
	perms *helper.PermissionSet,
) *PortfolioService {
	return &PortfolioService{portfolios: portfolios, students: students, perms: perms}
}

func translatePortfolioError(err error) error {
	switch {
	case errors.Is(err, repository.ErrInvalidID):
		return helper.BadRequest("id proyek tidak valid")
	case errors.Is(err, repository.ErrNotFound):
		return helper.NotFound("mahasiswa tidak ditemukan")
	case errors.Is(err, repository.ErrPortfolioNotFound):
		return helper.NotFound("portofolio belum dibuat, buat dengan PUT /students/:id/portfolio")
	case errors.Is(err, repository.ErrProjectNotFound):
		return helper.NotFound("proyek tidak ditemukan")
	case errors.Is(err, repository.ErrProjectLimit):
		return helper.NewError(fiber.StatusConflict, "PROJECT_LIMIT",
			"jumlah proyek sudah mencapai batas "+strconv.Itoa(repository.MaxProjectsPerPortfolio))
	default:
		return helper.Internal(err)
	}
}

// requireStudent memastikan mahasiswanya ada di PostgreSQL.
//
// Tidak ada FOREIGN KEY yang menghubungkan MongoDB dengan PostgreSQL, jadi
// pemeriksaan "mahasiswanya ada" harus dilakukan sendiri oleh service.
func (s *PortfolioService) requireStudent(ctx context.Context, c *fiber.Ctx) (int, *int, error) {
	studentID, valid := helper.ParamID(c)
	if !valid {
		return 0, nil, helper.BadRequest("id mahasiswa harus berupa angka positif")
	}
	ownerID, err := s.students.FindOwnerID(ctx, studentID)
	if err != nil {
		return 0, nil, translatePortfolioError(err)
	}
	return studentID, ownerID, nil
}

// requireWriteAccess = mahasiswa ada + pemanggil boleh mengubah datanya.
// Hak ubah: pemilik data mahasiswa itu, atau pemegang permission
// portfolio:manage (pola yang sama dengan CanAccessStudent pada Modul 6).
func (s *PortfolioService) requireWriteAccess(ctx context.Context, c *fiber.Ctx) (int, error) {
	current, ok := helper.CurrentUser(c)
	if !ok {
		return 0, helper.Unauthorized("belum terautentikasi")
	}
	studentID, ownerID, err := s.requireStudent(ctx, c)
	if err != nil {
		return 0, err
	}
	if !CanAccessStudent(current, ownerIDValue(ownerID), s.perms, "portfolio:manage") {
		return 0, helper.Forbidden("Anda tidak berhak mengubah portofolio mahasiswa ini")
	}
	return studentID, nil
}

func normalizeProject(in model.ProjectInput) model.ProjectInput {
	in.Title = strings.TrimSpace(in.Title)
	in.URL = strings.TrimSpace(in.URL)
	in.Technologies = NormalizeTechnologies(in.Technologies)
	return in
}

func projectFromInput(in model.ProjectInput) model.Project {
	return model.Project{
		Title: in.Title, Technologies: in.Technologies, URL: in.URL, Year: in.Year,
	}
}

// ---------- GET /students/:id/portfolio ----------
func (s *PortfolioService) Get(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, _, err := s.requireStudent(ctx, c)
	if err != nil {
		return err
	}
	portfolio, err := s.portfolios.FindByStudent(ctx, studentID)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio ditemukan", portfolio)
}

// ---------- PUT /students/:id/portfolio ----------
func (s *PortfolioService) Replace(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.requireWriteAccess(ctx, c)
	if err != nil {
		return err
	}

	// Body didekode ke struct bertipe tetap, BUKAN map[string]any. Bila
	// penyerang mengirim {"headline": {"$ne": ""}}, dekoder menolaknya
	// sebelum menyentuh database (lihat A.8 pada modul).
	var req model.PortfolioRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req.Headline = strings.TrimSpace(req.Headline)
	req.Bio = strings.TrimSpace(req.Bio)
	for i := range req.Projects {
		req.Projects[i] = normalizeProject(req.Projects[i])
	}
	for i := range req.Certificates {
		req.Certificates[i].Name = strings.TrimSpace(req.Certificates[i].Name)
		req.Certificates[i].Issuer = strings.TrimSpace(req.Certificates[i].Issuer)
		req.Certificates[i].Date = strings.TrimSpace(req.Certificates[i].Date)
	}
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	portfolio := model.Portfolio{
		Headline:     req.Headline,
		Bio:          req.Bio,
		Projects:     make([]model.Project, 0, len(req.Projects)),
		Certificates: make([]model.Certificate, 0, len(req.Certificates)),
	}
	for _, p := range req.Projects {
		portfolio.Projects = append(portfolio.Projects, projectFromInput(p))
	}
	for _, in := range req.Certificates {
		date, err := time.Parse("2006-01-02", in.Date)
		if err != nil {
			return helper.Validation(map[string]string{"date": "format tanggal harus YYYY-MM-DD"})
		}
		portfolio.Certificates = append(portfolio.Certificates, model.Certificate{
			Name: in.Name, Issuer: in.Issuer, Date: date,
		})
	}

	saved, err := s.portfolios.Replace(ctx, studentID, portfolio)
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Success(c, fiber.StatusOK, "portofolio berhasil disimpan", saved)
}

// ---------- POST /students/:id/portfolio/projects ----------
func (s *PortfolioService) AddProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.requireWriteAccess(ctx, c)
	if err != nil {
		return err
	}

	var req model.ProjectInput
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body harus berupa JSON yang valid")
	}
	req = normalizeProject(req)
	if errs := helper.ValidateStruct(req); errs != nil {
		return helper.Validation(errs)
	}

	created, err := s.portfolios.AddProject(ctx, studentID, projectFromInput(req))
	if err != nil {
		return translatePortfolioError(err)
	}
	return helper.Created(c, "proyek berhasil ditambahkan", created,
		"/api/v1/students/"+strconv.Itoa(studentID)+"/portfolio")
}

// ---------- DELETE /students/:id/portfolio/projects/:projectId ----------
func (s *PortfolioService) DeleteProject(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	studentID, err := s.requireWriteAccess(ctx, c)
	if err != nil {
		return err
	}
	if err := s.portfolios.DeleteProject(ctx, studentID, c.Params("projectId")); err != nil {
		return translatePortfolioError(err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ---------- GET /portfolios/stats/technologies ----------
func (s *PortfolioService) TechnologyStats(c *fiber.Ctx) error {
	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	stats, err := s.portfolios.TopTechnologies(ctx, 10)
	if err != nil {
		return helper.Internal(err)
	}
	return helper.Success(c, fiber.StatusOK, "10 teknologi terpopuler", stats)
}