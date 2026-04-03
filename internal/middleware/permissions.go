package middleware

import "github.com/Umairnoor2398/examify-backend/internal/models"

type Permission string

const (
	// Admin permissions
	PermAdminSchoolsRead   Permission = "admin:schools:read"
	PermAdminSchoolsCreate Permission = "admin:schools:create"
	PermAdminSchoolsUpdate Permission = "admin:schools:update"
	PermAdminSchoolsDelete Permission = "admin:schools:delete"

	PermAdminCurriculaRead   Permission = "admin:curricula:read"
	PermAdminCurriculaCreate Permission = "admin:curricula:create"
	PermAdminCurriculaUpdate Permission = "admin:curricula:update"
	PermAdminCurriculaDelete Permission = "admin:curricula:delete"

	PermAdminSubjectsRead   Permission = "admin:subjects:read"
	PermAdminSubjectsCreate Permission = "admin:subjects:create"
	PermAdminSubjectsUpdate Permission = "admin:subjects:update"
	PermAdminSubjectsDelete Permission = "admin:subjects:delete"

	PermAdminClassesRead   Permission = "admin:classes:read"
	PermAdminClassesCreate Permission = "admin:classes:create"
	PermAdminClassesUpdate Permission = "admin:classes:update"
	PermAdminClassesDelete Permission = "admin:classes:delete"

	PermAdminBooksRead   Permission = "admin:books:read"
	PermAdminBooksCreate Permission = "admin:books:create"
	PermAdminBooksUpdate Permission = "admin:books:update"
	PermAdminBooksDelete Permission = "admin:books:delete"
	PermAdminBooksImport Permission = "admin:books:import"

	PermAdminChaptersRead   Permission = "admin:chapters:read"
	PermAdminChaptersCreate Permission = "admin:chapters:create"
	PermAdminChaptersUpdate Permission = "admin:chapters:update"
	PermAdminChaptersDelete Permission = "admin:chapters:delete"
	PermAdminChaptersImport Permission = "admin:chapters:import"

	PermAdminQuestionsRead   Permission = "admin:questions:read"
	PermAdminQuestionsCreate Permission = "admin:questions:create"
	PermAdminQuestionsUpdate Permission = "admin:questions:update"
	PermAdminQuestionsDelete Permission = "admin:questions:delete"
	PermAdminQuestionsImport Permission = "admin:questions:import"

	// School-Admin permissions
	PermSchoolProfileRead   Permission = "school:profile:read"
	PermSchoolProfileUpdate Permission = "school:profile:update"

	PermSchoolTeachersRead   Permission = "school:teachers:read"
	PermSchoolTeachersCreate Permission = "school:teachers:create"
	PermSchoolTeachersUpdate Permission = "school:teachers:update"
	PermSchoolTeachersDelete Permission = "school:teachers:delete"

	PermSchoolBooksRead   Permission = "school:books:read"
	PermSchoolBooksCreate Permission = "school:books:create"
	PermSchoolBooksUpdate Permission = "school:books:update"
	PermSchoolBooksDelete Permission = "school:books:delete"

	PermSchoolPapersRead   Permission = "school:papers:read"
	PermSchoolPapersCreate Permission = "school:papers:create"
	PermSchoolPapersDelete Permission = "school:papers:delete"
	PermSchoolPapersPrint  Permission = "school:papers:print"

	// Teacher permissions
	PermTeacherBooksRead     Permission = "teacher:books:read"
	PermTeacherPapersRead    Permission = "teacher:papers:read"
	PermTeacherPapersCreate  Permission = "teacher:papers:create"
	PermTeacherPapersUpdate  Permission = "teacher:papers:update"
	PermTeacherPapersDelete  Permission = "teacher:papers:delete"
	PermTeacherPapersSubmit  Permission = "teacher:papers:submit"
)

var RolePermissions = map[models.Role][]Permission{
	models.RoleAdmin: {
		PermAdminSchoolsRead, PermAdminSchoolsCreate, PermAdminSchoolsUpdate, PermAdminSchoolsDelete,
		PermAdminCurriculaRead, PermAdminCurriculaCreate, PermAdminCurriculaUpdate, PermAdminCurriculaDelete,
		PermAdminSubjectsRead, PermAdminSubjectsCreate, PermAdminSubjectsUpdate, PermAdminSubjectsDelete,
		PermAdminClassesRead, PermAdminClassesCreate, PermAdminClassesUpdate, PermAdminClassesDelete,
		PermAdminBooksRead, PermAdminBooksCreate, PermAdminBooksUpdate, PermAdminBooksDelete, PermAdminBooksImport,
		PermAdminChaptersRead, PermAdminChaptersCreate, PermAdminChaptersUpdate, PermAdminChaptersDelete, PermAdminChaptersImport,
		PermAdminQuestionsRead, PermAdminQuestionsCreate, PermAdminQuestionsUpdate, PermAdminQuestionsDelete, PermAdminQuestionsImport,
	},
	models.RoleSchoolAdmin: {
		PermSchoolProfileRead, PermSchoolProfileUpdate,
		PermSchoolTeachersRead, PermSchoolTeachersCreate, PermSchoolTeachersUpdate, PermSchoolTeachersDelete,
		PermSchoolBooksRead, PermSchoolBooksCreate, PermSchoolBooksUpdate, PermSchoolBooksDelete,
		PermSchoolPapersRead, PermSchoolPapersCreate, PermSchoolPapersDelete, PermSchoolPapersPrint,
	},
	models.RoleTeacher: {
		PermTeacherBooksRead,
		PermTeacherPapersRead, PermTeacherPapersCreate, PermTeacherPapersUpdate, PermTeacherPapersDelete, PermTeacherPapersSubmit,
	},
}

func GetPermissions(role models.Role) []string {
	perms := RolePermissions[role]
	result := make([]string, len(perms))
	for i, p := range perms {
		result[i] = string(p)
	}
	return result
}
