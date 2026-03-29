package model

import (
	"database/sql/driver"
	"fmt"
)

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

type ConsentAction string

var ConsentActionConst = ConsentActionValues{
	Granted: "GRANTED",
	Revoked: "REVOKED",
}

type ConsentActionValues struct {
	Granted ConsentAction
	Revoked ConsentAction
}

func (ca ConsentAction) Value() string {
	return string(ca)
}

type EmergencyAccessStatus string

var (
	EmergencyAccessStatusConst = EmergencyAccessStatusValues{
		Requested: "REQUESTED",
		Approved:  "APPROVED",
		Rejected:  "REJECTED",
		Expired:   "EXPIRED",
	}
	MapEmergencyAccessStatus = map[string]EmergencyAccessStatus{
		"REQUESTED": EmergencyAccessStatusConst.Requested,
		"APPROVED":  EmergencyAccessStatusConst.Approved,
		"REJECTED":  EmergencyAccessStatusConst.Rejected,
		"EXPIRED":   EmergencyAccessStatusConst.Expired,
	}
)

type EmergencyAccessStatusValues struct {
	Requested EmergencyAccessStatus
	Approved  EmergencyAccessStatus
	Rejected  EmergencyAccessStatus
	Expired   EmergencyAccessStatus
}

func (r Role) Value() (driver.Value, error) {
	switch r {
	case RolesConst.Admin, RolesConst.Patient, RolesConst.Doctor, RolesConst.HoD:
		// valid role
	default:
		return nil, fmt.Errorf("invalid role: %s", r)
	}
	return string(r), nil
}

func (r *Role) Scan(src interface{}) error {
	if src == nil {
		*r = ""
		return nil
	}

	s, ok := src.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into Role", src)
	}

	switch s {
	case string(RolesConst.Admin), string(RolesConst.Patient), string(RolesConst.Doctor), string(RolesConst.HoD):
		// valid role
	default:
		return fmt.Errorf("invalid role: %s", s)
	}

	*r = Role(s)
	return nil
}
