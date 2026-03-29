export type UserRole = "PATIENT" | "DOCTOR" | "HOD";

export const ROLE_HOME: Record<UserRole, string> = {
    PATIENT: "/dashboard",
    DOCTOR: "/doctor",
    HOD: "/hod",
};

export const ROLE_ALLOWED_PATHS: Record<UserRole, string[]> = {
    PATIENT: [
        "/patient",
        "/patient/bundles",
        "/patient/consents",
        "/bundles",
    ],

    DOCTOR: [
        "/doctor",
        "/doctor/bundles",
        "/doctor/consents",
        "/patient/consents",
        "/doctor/break-glass",
        "/bundles",
        "/break-glass"
    ],

    HOD: [
        "/hod",
        "/hod/break-glass",
        "/break-glass"
    ],
};