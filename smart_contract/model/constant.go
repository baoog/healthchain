package model

var RolesConst = Roles{
	Admin:   "ADMIN",
	Patient: "PATIENT",
	Doctor:  "DOCTOR",
	HoD:     "HOD",
}

type Role string

type Roles struct {
	Admin   Role
	Patient Role
	Doctor  Role
	HoD     Role
}
