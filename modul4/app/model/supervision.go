package model

type Supervisor struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Quota          int    `json:"quota"`
	SupervisedCount int   `json:"supervised_count"`
	Version        int    `json:"version"`
}

type AssignSupervisorRequest struct {
	StudentID int `json:"student_id" validate:"required,min=1"`
}

type SwitchSupervisorRequest struct {
	StudentID   int `json:"student_id" validate:"required,min=1"`
	SupervisorID int `json:"supervisor_id" validate:"required,min=1"`
}

type UpdateSupervisorQuotaRequest struct {
	Quota   int `json:"quota" validate:"required,min=1,max=500"`
	Version int `json:"version" validate:"required,min=1"`
}