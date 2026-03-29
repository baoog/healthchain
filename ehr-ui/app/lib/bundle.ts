export type BundleType =
    | "ALLERGY"
    | "MEDICATION"
    | "BLOOD_GROUP"
    | "EMERGENCY_SUMMARY";

export interface BundleResponse<T = any> {
    bundle: Bundle;
    data: T[];
    ok: boolean;
}

export interface Bundle {
    id: string;
    owner_hash: string;
    type: BundleType;
    uri:string;
    archetype_id: string;
    created_at: string;
    is_external: boolean;
}

// ---- Content types ----

export interface AllergyData {
    allergen: string;
    reaction: string;
    severity: string;
    onsetDate?: string;
    notes?: string;
}

export interface MedicationData {
    drug_name: string;
    dose: string;
    frequency: string;
    route: string;
    start_date?: string;
    end_date?: string;
    notes?: string;
}

export interface BloodGroupData {
    abo: string;
    rh: string;
    last_confirmed?: string;
}

export interface EmergencySummaryData {
    problems: string[];
    allergies?: string[];
    current_medications?: string[];
    notes?: string;
}